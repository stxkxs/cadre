/**
 * Schema barrel export.
 *
 * - shared.ts: public schema tables (users, integrationConnections)
 *   shared across all sibling apps
 * - cadre.ts: cadre-namespaced tables (pgSchema('cadre'))
 *   isolated to this application
 */

export { users, engagements, integrationConnections } from './shared';
export {
  cadreSchema,
  workflows,
  runs,
} from './cadre';
