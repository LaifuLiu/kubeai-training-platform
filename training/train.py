import json
import os
import random
import time

import torch
from torch import nn

SEED = int(os.getenv("SEED", "42"))
EPOCHS = int(os.getenv("EPOCHS", "100"))
LEARNING_RATE = float(os.getenv("LEARNING_RATE", "0.03"))

random.seed(SEED)
torch.manual_seed(SEED)
torch.set_num_threads(2)

def log(event, **kwargs):
    print(
        json.dumps({"event": event, **kwargs}),
        flush=True,
    )

def main():
    # Synthetic dataset: y = 2x + 3 + noise
    x = torch.linspace(-10, 10, 1000).reshape(-1, 1)
    noise = torch.randn_like(x) * 0.5
    y = 2 * x + 3 + noise

    model = nn.Linear(1, 1)
    optimizer = torch.optim.SGD(
        model.parameters(),
        lr=LEARNING_RATE,
    )
    loss_fn = nn.MSELoss()

    log(
        "training_started",
        seed=SEED,
        epochs=EPOCHS,
        samples=len(x),
    )

    start = time.time()

    for epoch in range(1, EPOCHS + 1):
        prediction = model(x)
        loss = loss_fn(prediction, y)

        optimizer.zero_grad()
        loss.backward()
        optimizer.step()

        if epoch == 1 or epoch % 10 == 0 or epoch == EPOCHS:
            log(
                "training_progress",
                epoch=epoch,
                loss=round(loss.item(), 6),
            )

    weight = model.weight.item()
    bias = model.bias.item()

    log(
        "training_completed",
        weight=round(weight, 4),
        bias=round(bias, 4),
        duration_seconds=round(time.time() - start, 3),
    )

if __name__ == "__main__":
    main()
