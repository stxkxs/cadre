import NextAuth from 'next-auth';
import GitHub from 'next-auth/providers/github';
import { db } from '@/lib/db';
import { users } from '@/lib/db/schema';
import { eq } from 'drizzle-orm';
import { logger } from '@/lib/logger';

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [
    GitHub({
      clientId: process.env.AUTH_GITHUB_ID!,
      clientSecret: process.env.AUTH_GITHUB_SECRET!,
    }),
  ],
  pages: {
    signIn: '/login',
  },
  trustHost: true,
  debug: process.env.NODE_ENV === 'development',
  callbacks: {
    async signIn({ user, profile }) {
      if (!user.email) return false;
      try {
        // Upsert user in database on sign-in
        const githubId = String((profile as { id?: number })?.id || '');
        const existing = await db
          .select({ id: users.id })
          .from(users)
          .where(eq(users.email, user.email))
          .limit(1);

        if (existing.length === 0) {
          await db.insert(users).values({
            email: user.email,
            name: user.name || null,
            image: user.image || null,
            githubId,
          });
        } else {
          // Update profile info on each login
          await db
            .update(users)
            .set({ name: user.name || null, image: user.image || null, githubId })
            .where(eq(users.email, user.email));
        }
      } catch (error) {
        logger.error('Failed to upsert user', { error: String(error) });
        return false;
      }
      return true;
    },
    async session({ session, token }) {
      if (session.user && token.dbId) {
        session.user.id = token.dbId as string;
      }
      return session;
    },
    async jwt({ token, user, trigger }) {
      // On initial sign-in or whenever we don't have the DB id yet, look it up
      if ((trigger === 'signIn' || !token.dbId) && (user?.email || token.email)) {
        try {
          const email = user?.email || (token.email as string);
          const [dbUser] = await db
            .select({ id: users.id })
            .from(users)
            .where(eq(users.email, email))
            .limit(1);
          if (dbUser) {
            token.dbId = dbUser.id;
          }
        } catch (error) {
          logger.error('Failed to look up user', { error: String(error) });
        }
      }
      return token;
    },
  },
  session: {
    strategy: 'jwt',
  },
});
