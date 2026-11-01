// Package workflow shares advisory PR review eligibility, revision-pinned source context, inference,
// result identity, and safe publication. It refreshes consent through
// commands.Authorizer. Features supply their own prompts, model configuration,
// and independently registered queue jobs; replies use one shared template.
// Enabled jobs also reconcile their own outdated inline threads without inference.
package workflow
