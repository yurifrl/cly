package incidentsio

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type renderer interface {
	Render(bundles []IncidentBundle) error
}

func newRenderer(opts Options) renderer {
	switch opts.Format {
	case "json":
		return &jsonRenderer{writer: os.Stdout}
	case "text":
		return &textRenderer{writer: os.Stdout}
	case "charm":
		return &charmRenderer{writer: os.Stdout}
	case "markdown":
		return &markdownRenderer{writer: os.Stdout}
	default:
		return &markdownRenderer{writer: os.Stdout}
	}
}

type jsonRenderer struct {
	writer io.Writer
}

func (r *jsonRenderer) Render(bundles []IncidentBundle) error {
	enc := json.NewEncoder(r.writer)
	enc.SetIndent("", "  ")
	return enc.Encode(bundles)
}

type textRenderer struct {
	writer io.Writer
}

func (r *textRenderer) Render(bundles []IncidentBundle) error {
	for _, bundle := range bundles {
		if _, err := fmt.Fprintf(r.writer, "• %s [%s] — %s\n",
			bundle.Incident.Reference,
			bundle.Incident.Status.Name,
			bundle.Incident.Name); err != nil {
			return err
		}
	}
	return nil
}

type charmRenderer struct {
	writer io.Writer
}

func (r *charmRenderer) Render(bundles []IncidentBundle) error {
	if _, err := fmt.Fprintln(r.writer, "🔥 Incidents List"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(r.writer, "==============="); err != nil {
		return err
	}

	for _, bundle := range bundles {
		if _, err := fmt.Fprintf(r.writer, "• %s [%s] — %s\n",
			bundle.Incident.Reference,
			bundle.Incident.Status.Name,
			bundle.Incident.Name); err != nil {
			return err
		}
	}
	return nil
}

type markdownRenderer struct {
	writer io.Writer
}

func (r *markdownRenderer) Render(bundles []IncidentBundle) error {
	for _, bundle := range bundles {
		if _, err := fmt.Fprintf(r.writer, "# %s %s @%s\n\n",
			bundle.Incident.Reference,
			bundle.Incident.Name,
			bundle.Leader.Name); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(r.writer, "## Follow-ups"); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(r.writer, "Created at: %s\n", bundle.Incident.CreatedAt.Format("2006-01-02")); err != nil {
			return err
		}

		for _, followup := range bundle.Followups {
			checked := " "
			if followup.Status == "completed" {
				checked = "x"
			}

			assigneeName := "Unassigned"
			if followup.Assignee.Name != "" {
				assigneeName = followup.Assignee.Name
			}

			if _, err := fmt.Fprintf(r.writer, "- [%s] %s @%s\n",
				checked,
				followup.Title,
				assigneeName); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintln(r.writer); err != nil {
			return err
		}
	}

	return nil
}
