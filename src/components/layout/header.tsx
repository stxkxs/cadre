'use client';

import React, { useEffect, useState } from 'react';
import { Moon, Sun } from 'lucide-react';
import { useSettingsStore } from '@/lib/store/settings-store';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';

interface UserInfo {
  name: string | null;
  email: string | null;
  image: string | null;
}

export function Header() {
  const { theme, setTheme } = useSettingsStore();
  const [user, setUser] = useState<UserInfo | null>(null);

  useEffect(() => {
    fetch('/api/auth/session')
      .then(res => res.json())
      .then(data => {
        if (data?.user) setUser(data.user);
      })
      .catch(() => {});
  }, []);

  const initials = user?.name
    ? user.name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
    : user?.email?.[0]?.toUpperCase() || '?';

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-frosted backdrop-blur-xl px-6">
      <div />
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          className="text-dim hover:text-foreground"
          aria-label="Toggle theme"
        >
          {theme === 'dark' ? (
            <Sun className="h-5 w-5" />
          ) : (
            <Moon className="h-5 w-5" />
          )}
        </Button>
        {user && (
          <div className="flex items-center gap-2">
            <Avatar className="h-8 w-8">
              {user.image && <AvatarImage src={user.image} alt={user.name || ''} />}
              <AvatarFallback className="bg-[#0AEFB7] text-[#06090F] text-xs font-semibold">{initials}</AvatarFallback>
            </Avatar>
            {user.name && (
              <span className="text-sm text-muted-foreground hidden md:inline">{user.name}</span>
            )}
          </div>
        )}
      </div>
    </header>
  );
}
