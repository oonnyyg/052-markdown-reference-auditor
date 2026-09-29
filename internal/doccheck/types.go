package doccheck

// LinkStatus is the result class of checking a local link.
type LinkStatus string

const (
	StatusOK              LinkStatus = "ok"
	StatusOutsideRoot     LinkStatus = "outside_root"
	StatusFileMissing     LinkStatus = "file_missing"
	StatusIllegalTarget   LinkStatus = "illegal_target"
	StatusAnchorMissing   LinkStatus = "anchor_missing"
)

// LinkFinding describes one inline link occurrence in source order.
type LinkFinding struct {
	Source string     `json:"source"`
	Line   int        `json:"line"`
	Order  int        `json:"order"`
	Target string     `json:"target"`
	Path   string     `json:"path"`
	Anchor string     `json:"anchor,omitempty"`
	Status LinkStatus `json:"status"`
	Detail string     `json:"detail,omitempty"`
}

// PageReport is the full JSON report document.
type PageReport struct {
	Root        string        `json:"root"`
	Entry       string        `json:"entry"`
	Links       []LinkFinding `json:"links"`
	Unreachable []string      `json:"unreachable"`
	Counts      Counts        `json:"counts"`
}

// Counts groups findings by status.
type Counts struct {
	TotalLinks    int `json:"total_links"`
	OK            int `json:"ok"`
	OutsideRoot   int `json:"outside_root"`
	FileMissing   int `json:"file_missing"`
	IllegalTarget int `json:"illegal_target"`
	AnchorMissing int `json:"anchor_missing"`
	Unreachable   int `json:"unreachable"`
}

// Result is the in-memory audit outcome.
type Result struct {
	Report PageReport
	// Unreachable holds slash-style relative page paths.
	Unreachable []string
}

func (r *Result) HasProblem() bool {
	c := r.Report.Counts
	return c.OutsideRoot+c.FileMissing+c.IllegalTarget+c.AnchorMissing+c.Unreachable > 0
}

func (r *Result) ProblemCount() int {
	c := r.Report.Counts
	return c.OutsideRoot + c.FileMissing + c.IllegalTarget + c.AnchorMissing
}
