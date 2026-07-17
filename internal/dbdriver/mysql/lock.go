// lock.go: the Locker facet.
//
// GET_LOCK is session-scoped, so AcquireLock runs on the dedicated
// connection created at Driver construction — one that is never returned
// to the pool and is used for nothing else. Fail-fast: GET_LOCK with a
// zero timeout; unavailable maps to dbdriver.ErrLockHeld. release frees
// the lock and closes the session.
package mysql
