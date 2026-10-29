
import json
import os
import re
import sys

import torch
import torch.distributed as dist
from torch import nn
from torch.nn.parallel import DistributedDataParallel as DDP
from torch.utils.data import DataLoader, TensorDataset
from torch.utils.data.distributed import DistributedSampler


def get_rank(world_size: int) -> int:
    """Get the worker rank from torchrun or the Kubernetes Pod name."""
    rank_env = os.getenv("RANK")

    if rank_env is not None:
        return int(rank_env)

    if world_size <= 1:
        return 0

    pod_name = os.getenv("POD_NAME", "")
    match = re.search(r"-trainer-(\d+)$", pod_name)

    if not match:
        raise RuntimeError(
            f"Cannot determine distributed rank from POD_NAME={pod_name!r}. "
            "Expected a name ending in '-trainer-<index>', "
            "or set RANK explicitly."
        )

    return int(match.group(1))


def log_event(event: str, **fields) -> None:
    """Emit machine-readable JSON logs."""
    print(
        json.dumps({"event": event, **fields}),
        flush=True,
    )


def main() -> None:
    epochs = int(os.getenv("EPOCHS", "100"))
    seed = int(os.getenv("SEED", "42"))
    world_size = int(os.getenv("WORLD_SIZE", "1"))
    batch_size = int(os.getenv("BATCH_SIZE", "64"))
    learning_rate = float(os.getenv("LEARNING_RATE", "0.01"))

    if epochs < 1:
        raise ValueError("EPOCHS must be at least 1")
    if world_size < 1:
        raise ValueError("WORLD_SIZE must be at least 1")
    if batch_size < 1:
        raise ValueError("BATCH_SIZE must be at least 1")

    rank = get_rank(world_size)

    if not 0 <= rank < world_size:
        raise ValueError(
            f"RANK={rank} must be between 0 and WORLD_SIZE-1 "
            f"(WORLD_SIZE={world_size})"
        )

    distributed = world_size > 1
    process_group_initialized = False

    try:
        # Initialize distributed communication when multiple workers are used.
        if distributed:
            master_addr = os.getenv("MASTER_ADDR")
            master_port = os.getenv("MASTER_PORT", "29500")

            if not master_addr:
                raise RuntimeError(
                    "MASTER_ADDR is required for distributed training"
                )

            os.environ["RANK"] = str(rank)
            os.environ["WORLD_SIZE"] = str(world_size)
            os.environ["MASTER_PORT"] = master_port

            dist.init_process_group(
                backend="gloo",
                init_method="env://",
                rank=rank,
                world_size=world_size,
            )
            process_group_initialized = True

        # Generate the same dataset on each worker.
        # DistributedSampler assigns different samples to different workers.
        generator = torch.Generator().manual_seed(seed)

        x = torch.randn(4096, 1, generator=generator)
        noise = 0.1 * torch.randn(4096, 1, generator=generator)
        y = 2.0 * x + 3.0 + noise

        dataset = TensorDataset(x, y)

        sampler = None
        if distributed:
            sampler = DistributedSampler(
                dataset,
                num_replicas=world_size,
                rank=rank,
                shuffle=True,
                seed=seed,
                drop_last=False,
            )

        loader = DataLoader(
            dataset,
            batch_size=batch_size,
            sampler=sampler,
            shuffle=(sampler is None),
            drop_last=False,
            num_workers=0,
        )

        # Set the same initial seed on each worker.
        torch.manual_seed(seed)

        model = nn.Linear(1, 1)
        if distributed:
            model = DDP(model)

        optimizer = torch.optim.SGD(
            model.parameters(),
            lr=learning_rate,
        )
        criterion = nn.MSELoss()

        if rank == 0:
            log_event(
                "training_started",
                epochs=epochs,
                world_size=world_size,
                distributed=distributed,
                dataset_size=len(dataset),
                batch_size=batch_size,
            )

        for epoch in range(epochs):
            if sampler is not None:
                sampler.set_epoch(epoch)

            model.train()
            total_loss = 0.0
            total_samples = 0

            for batch_x, batch_y in loader:
                optimizer.zero_grad(set_to_none=True)

                predictions = model(batch_x)
                loss = criterion(predictions, batch_y)

                loss.backward()

                # DDP synchronizes gradients during backward().
                # Log the first batch from each rank to prove participation.
                if epoch == 0 and total_samples == 0:
                    log_event(
                        "worker_training_step",
                        rank=rank,
                        world_size=world_size,
                        batch_size=batch_x.size(0),
                        loss=round(loss.item(), 6),
                        weight_grad=round(model.module.weight.grad.item(), 6)
                        if distributed
                        else round(model.weight.grad.item(), 6),
                    )

                optimizer.step()

                sample_count = batch_x.size(0)
                total_loss += loss.item() * sample_count
                total_samples += sample_count

                sample_count = batch_x.size(0)
                total_loss += loss.item() * sample_count
                total_samples += sample_count

            average_loss = total_loss / total_samples

            if rank == 0:
                log_event(
                    "training_progress",
                    epoch=epoch + 1,
                    epochs=epochs,
                    loss=round(average_loss, 6),
                )

        # DDP keeps model parameters synchronized across workers.
        if distributed:
            dist.barrier()

        if rank == 0:
            trained_model = model.module if distributed else model

            weight = trained_model.weight.item()
            bias = trained_model.bias.item()

            log_event(
                "training_completed",
                rank=rank,
                world_size=world_size,
                weight=round(weight, 6),
                bias=round(bias, 6),
            )

    except Exception as exc:
        print(
            f"Training failed on rank {rank}: {exc}",
            file=sys.stderr,
            flush=True,
        )
        raise

    finally:
        if process_group_initialized:
            dist.destroy_process_group()


if __name__ == "__main__":
    main()
