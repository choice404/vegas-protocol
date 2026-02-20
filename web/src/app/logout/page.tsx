import Link from "next/link";
import { redirect } from "next/navigation";

import { auth, signOut } from "@/server/auth";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";

export default async function LogoutPage() {
  const session = await auth();
  if (!session) redirect("/");

  return (
    <div className="bg-background flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Sign out</CardTitle>
          <CardDescription>
            Are you sure you want to sign out?
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <form
            action={async () => {
              "use server";
              await signOut({ redirectTo: "/" });
            }}
          >
            <Button type="submit" variant="destructive" className="w-full">
              Sign Out
            </Button>
          </form>
          <Button asChild variant="outline" className="w-full">
            <Link href="/">Cancel</Link>
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
