// Package workflow coordinates admission, inference, and publication for
// independent review commands. It rechecks human consent and PR revisions before
// sending source and publishing results. One factory owns the publication gate,
// which serializes final history checks and submission across features.
// Manual requests wake their own job. Scheduled runs also reconcile outdated
// inline threads after draining the manual queue.
// Reviews and thread maintenance require both branches in the target repository;
// fork pull requests are ignored.
package workflow
