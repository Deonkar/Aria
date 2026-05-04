'use client'

export function AriaLogo({ className = "" }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 120 120"
      className={className}
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <linearGradient id="ariaGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#65a3d5" />
          <stop offset="100%" stopColor="#a5a3d5" />
        </linearGradient>
        <linearGradient id="accentGradient" x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%" stopColor="#ff006e" />
          <stop offset="50%" stopColor="#65a3d5" />
          <stop offset="100%" stopColor="#ffd60a" />
        </linearGradient>
      </defs>

      {/* Background circle */}
      <circle cx="60" cy="60" r="58" fill="url(#ariaGradient)" opacity="0.1" />

      {/* Main icon - Abstract AI wave */}
      <g transform="translate(30, 35)">
        {/* Curved waves representing AI */}
        <path
          d="M 10 30 Q 15 20, 20 30 T 40 30"
          stroke="url(#ariaGradient)"
          strokeWidth="2.5"
          fill="none"
          strokeLinecap="round"
        />
        <path
          d="M 5 20 Q 15 8, 25 20 T 45 20"
          stroke="url(#ariaGradient)"
          strokeWidth="2"
          fill="none"
          strokeLinecap="round"
          opacity="0.7"
        />
        <path
          d="M 15 40 Q 20 32, 30 40 T 50 40"
          stroke="url(#ariaGradient)"
          strokeWidth="2"
          fill="none"
          strokeLinecap="round"
          opacity="0.7"
        />

        {/* Center dot */}
        <circle cx="27.5" cy="30" r="1.5" fill="url(#ariaGradient)" />
      </g>

      {/* Accent pulse ring */}
      <circle
        cx="60"
        cy="60"
        r="48"
        fill="none"
        stroke="url(#ariaGradient)"
        strokeWidth="1"
        opacity="0.3"
      />
    </svg>
  )
}
