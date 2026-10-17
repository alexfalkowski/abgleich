package automerge

// PullRequest is a remote snapshot used to decide whether its head needs a
// rebase. ID identifies the pull request in remote mutations; Number identifies
// it in repository reads. AutoMerge records whether automatic merging is enabled.
type PullRequest struct {
	ID        string
	State     string
	Head      Branch
	Base      Branch
	Number    int
	Draft     bool
	AutoMerge bool
}

// Branch identifies a branch and the commit observed when it was read.
// Repository is the full repository name, such as "alexfalkowski/infraops".
type Branch struct {
	Repository string
	Name       string
	SHA        string
}

// Comparison records how many commits from the base are missing from the head.
type Comparison struct {
	BehindBy int
}
