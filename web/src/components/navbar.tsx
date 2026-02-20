import Link from "next/link";
import type { Session } from "next-auth";

import { Button } from "@/components/ui/button";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/components/ui/avatar";
import { ThemeToggle } from "@/components/theme-toggle";

export function Navbar({ session }: { session: Session | null }) {
  return (
    <header className="border-border border-b">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
        <Link
          href="/"
          className="text-foreground text-lg font-semibold tracking-tight"
        >
          Rebel Hacks
        </Link>

        <nav className="flex items-center gap-2">
          <ThemeToggle />
          {session ? (
            <div className="flex items-center gap-3">
              <Avatar size="sm">
                {session.user?.image && (
                  <AvatarImage
                    src={session.user.image}
                    alt={session.user.name ?? "User"}
                  />
                )}
                <AvatarFallback>
                  {session.user?.name?.charAt(0)?.toUpperCase() ?? "U"}
                </AvatarFallback>
              </Avatar>
              <span className="text-foreground text-sm">
                {session.user?.name}
              </span>
              <Button asChild variant="ghost" size="sm">
                <Link href="/logout">Sign Out</Link>
              </Button>
            </div>
          ) : (
            <Button asChild variant="ghost" size="sm">
              <Link href="/login">Sign In</Link>
            </Button>
          )}
        </nav>
      </div>
    </header>
  );
}
