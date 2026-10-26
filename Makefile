include bin/build/make/help.mak
include bin/build/make/http.mak
include bin/build/make/git.mak
include bin/build/make/claude.mak
include bin/build/make/codex.mak

# Run Dependabot and labeled merge policy features.
features-merges:
	@$(MAKE) features tags="@merges"

# Run Dependabot merge policy features.
features-dependabot:
	@$(MAKE) features tags="@dependabot"

# Run labeled merge policy features.
features-labels:
	@$(MAKE) features tags="@labels"

# Run automerge and status comment command features.
features-commands:
	@$(MAKE) features tags="@commands and not (@summaries or @reactions)"

# Run summary and comment reaction features.
features-summaries:
	@$(MAKE) features tags="@summaries or @reactions"

# Run reconciliation, recovery, scheduling, and shared installation features.
features-reconciliation:
	@$(MAKE) features tags="@reconciliation"

# Run the remaining worker, configuration, and health features.
features-worker:
	@$(MAKE) features tags="not (@merges or @commands or @summaries or @reconciliation)"

# Stage coverage profiles in an isolated CircleCI workspace partition (set partition to the feature group).
coverage-workspace-stage:
	@./scripts/coverage-workspace stage "$(partition)"

# Restore staged CircleCI coverage profiles to test/reports for merging.
coverage-workspace-restore:
	@./scripts/coverage-workspace restore
