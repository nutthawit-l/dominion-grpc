import { useEffect, useRef } from 'react'
import type { LogEntry } from '../store'

interface Props {
  entries: LogEntry[]
}

export default function GameLog({ entries }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  return (
    <div data-testid="game-log" className="flex flex-col gap-0.5 text-xs font-mono text-sol-base00">
      <p className="text-sol-base1 uppercase tracking-widest text-[10px] mb-1">Log</p>
      {entries.map((e) => (
        <div key={String(e.seq)} className="leading-relaxed">
          {e.text}
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  )
}
