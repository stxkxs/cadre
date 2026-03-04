import type { Metadata } from 'next';
import { JetBrains_Mono, IBM_Plex_Sans } from 'next/font/google';
import './globals.css';
import { AppShell } from '@/components/layout/app-shell';

const jetbrainsMono = JetBrains_Mono({
  subsets: ['latin'],
  variable: '--font-display',
  display: 'swap',
});

const ibmPlexSans = IBM_Plex_Sans({
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  variable: '--font-body',
  display: 'swap',
});

export const metadata: Metadata = {
  title: {
    default: 'Cadre - Agent Orchestrator',
    template: '%s | Cadre',
  },
  description: 'Autonomous agent workflow orchestrator with graph-based visual editor',
  metadataBase: new URL(process.env.AUTH_URL || process.env.NEXTAUTH_URL || 'http://localhost:3000'),
  openGraph: {
    title: 'Cadre - Agent Orchestrator',
    description: 'Build and run autonomous AI agent workflows with a visual graph editor',
    type: 'website',
  },
  robots: {
    index: false,
    follow: false,
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`dark ${jetbrainsMono.variable} ${ibmPlexSans.variable}`} suppressHydrationWarning>
      <body className="min-h-screen antialiased font-body" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
        <div id="app-root"><AppShell>{children}</AppShell></div>
      </body>
    </html>
  );
}
