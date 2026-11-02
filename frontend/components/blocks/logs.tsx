import {
  CodeBlock,
  CodeBlockActions,
  CodeBlockContent,
  CodeBlockCopyButton,
  CodeBlockHeader,
  CodeBlockTitle,
} from "@/components/ui/code-block";

import { useLogs } from "@/hooks/use-logs";

export function Logs({ slug }: { slug: string }) {
  const { data } = useLogs({ slug });

  return (
    <div className="w-full max-w-md">
      <CodeBlock>
        <CodeBlockHeader>
          <CodeBlockTitle>Logs</CodeBlockTitle>
          <CodeBlockActions>
            <CodeBlockCopyButton value={data ?? ""} />
          </CodeBlockActions>
        </CodeBlockHeader>
        <CodeBlockContent className="max-h-48 scroll-fade-y">
          {data ?? ""}
        </CodeBlockContent>
      </CodeBlock>
    </div>
  );
}
