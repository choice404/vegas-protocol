import {
  createTRPCRouter,
  protectedProcedure,
  publicProcedure,
} from "@/server/api/trpc";

// Rename this file and router once you know what you're building.
// Add your tRPC procedures here.

export const postRouter = createTRPCRouter({
  // Public health check — useful for verifying tRPC works
  health: publicProcedure.query(() => {
    return { status: "ok" };
  }),

  // Protected example — only accessible when signed in
  me: protectedProcedure.query(({ ctx }) => {
    return {
      id: ctx.session.user.id,
      name: ctx.session.user.name,
    };
  }),
});
