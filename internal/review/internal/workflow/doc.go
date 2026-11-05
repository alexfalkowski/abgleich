// Package workflow shares PR review eligibility, revision-pinned source context, inference,
// result identity, verdict selection, and safe publication. It refreshes consent through
// commands.Authorizer. Features supply their own prompts, model configuration,
// and independently registered queue jobs; replies use one shared template.
// Manual requests wake their job without repository-wide discovery. Scheduled
// runs also reconcile their own outdated inline threads without inference.
// One factory constructs independent commands sharing source access and serialized
// publication so an approval preserves the other feature's findings.
// Reviews and thread maintenance require both branches in the target repository;
// fork pull requests are ignored.
package workflow
