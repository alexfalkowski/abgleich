include bin/build/make/help.mak
include bin/build/make/http.mak
include bin/build/make/git.mak
include bin/build/make/claude.mak
include bin/build/make/codex.mak

# Run Dependabot and labeled merge policy features.
features-merges:
	@$(MAKE) features tags="@merges"

# Run comment command and reaction features.
features-commands:
	@$(MAKE) features tags="@commands"

# Run the remaining worker, configuration, and health features.
features-worker:
	@$(MAKE) features tags="not (@merges or @commands)"

# Stage coverage profiles in an isolated CircleCI workspace partition (partition=merges|commands|worker).
coverage-workspace-stage:
	@./scripts/coverage-workspace stage "$(partition)"

# Restore staged CircleCI coverage profiles to test/reports for merging.
coverage-workspace-restore:
	@./scripts/coverage-workspace restore
