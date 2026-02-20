import Link from "next/link";

import { auth } from "@/server/auth";
import { Button } from "@/components/ui/button";
import { Navbar } from "@/components/navbar";

export default async function Page() {
  const session = await auth();

  return (
    <div className="bg-background flex min-h-screen flex-col">
      <Navbar session={session} />

      <main className="flex flex-1 flex-col items-center justify-center px-4">
        <div className="flex max-w-2xl flex-col items-center gap-8 text-center">
          <div className="flex flex-col items-center gap-3">
            <h1 className="text-foreground text-5xl font-bold tracking-tight sm:text-6xl">
              Rebel Hacks
            </h1>
            <p className="text-muted-foreground max-w-md text-lg">
              Hackathon project starter. Sign in to get started.
            </p>
          </div>

          <div className="flex gap-3">
            {session ? (
              <Button asChild size="lg">
                <Link href="/dashboard">Go to Dashboard</Link>
              </Button>
            ) : (
              <Button asChild size="lg">
                <Link href="/login">Sign In</Link>
              </Button>
            )}
          </div>
        </div>
      </main>

      <footer className="text-muted-foreground border-border border-t py-6 text-center text-sm">
        Rebel Hacks
      </footer>
    </div>
  );
}
