package workflowtemplate

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	yaml "gopkg.in/yaml.v2"
)

func prepareParams(wt *sdk.WorkflowTemplate, r sdk.WorkflowTemplateRequest) interface{} {
	m := map[string]interface{}{}
	for _, p := range wt.Parameters {
		v, ok := r.Parameters[p.Key]
		if ok {
			switch p.Type {
			case sdk.ParameterTypeBoolean:
				m[p.Key] = v == "true"
			default:
				m[p.Key] = v
			}
		}
	}
	return m
}

func executeTemplate(t string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New(fmt.Sprintf("template")).Delims("[[", "]]").Parse(t)
	if err != nil {
		return "", sdk.WrapError(err, "cannot parse workflow template")
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", sdk.WrapError(err, "cannot execute workflow template")
	}

	return buffer.String(), nil
}

// Execute returns yaml file from template.
func Execute(wt *sdk.WorkflowTemplate, r sdk.WorkflowTemplateRequest) (sdk.WorkflowTemplateResult, error) {
	data := map[string]interface{}{
		"id":     wt.ID,
		"name":   r.Name,
		"params": prepareParams(wt, r),
	}

	v, err := base64.StdEncoding.DecodeString(wt.Value)
	if err != nil {
		return sdk.WorkflowTemplateResult{}, sdk.WrapError(err, "cannot parse workflow template")
	}

	out, err := executeTemplate(string(v), data)
	if err != nil {
		return sdk.WorkflowTemplateResult{}, err
	}

	res := sdk.WorkflowTemplateResult{
		Workflow:     out,
		Pipelines:    make([]string, len(wt.Pipelines)),
		Applications: make([]string, len(wt.Applications)),
	}

	for i, p := range wt.Pipelines {
		v, err := base64.StdEncoding.DecodeString(p.Value)
		if err != nil {
			return sdk.WorkflowTemplateResult{}, sdk.WrapError(err, "cannot parse pipeline template")
		}

		out, err := executeTemplate(string(v), data)
		if err != nil {
			return sdk.WorkflowTemplateResult{}, err
		}
		res.Pipelines[i] = out
	}

	for i, p := range wt.Applications {
		v, err := base64.StdEncoding.DecodeString(p.Value)
		if err != nil {
			return sdk.WorkflowTemplateResult{}, sdk.WrapError(err, "cannot parse application template")
		}

		out, err := executeTemplate(string(v), data)
		if err != nil {
			return sdk.WorkflowTemplateResult{}, err
		}
		res.Applications[i] = out
	}

	return res, nil
}

// Tar returns in buffer the a tar file that contains all generated stuff in template result.
func Tar(res sdk.WorkflowTemplateResult, w io.Writer) error {
	tw := tar.NewWriter(w)

	// add generated workflow to writer
	var wor exportentities.Workflow
	if err := yaml.Unmarshal([]byte(res.Workflow), &wor); err != nil {
		return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Cannot parse generated workflow"))
	}

	bs, err := exportentities.Marshal(wor, exportentities.FormatYAML)
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: fmt.Sprintf("%s.yml", wor.Name),
		Mode: 0644,
		Size: int64(len(bs)),
	}); err != nil {
		tw.Close()
		return sdk.WrapError(err, "Unable to write header for workflow %s", wor.Name)
	}
	if _, err := io.Copy(tw, bytes.NewBuffer(bs)); err != nil {
		tw.Close()
		return sdk.WrapError(err, "Unable to copy workflow buffer")
	}

	// add generated pipelines to writer
	for _, p := range res.Pipelines {
		var pip exportentities.PipelineV1
		if err := yaml.Unmarshal([]byte(p), &pip); err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Cannot parse generated pipeline"))
		}

		bs, err := exportentities.Marshal(pip, exportentities.FormatYAML)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf("%s.pip.yml", pip.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			tw.Close()
			return sdk.WrapError(err, "Unable to write header for pipeline %s", pip.Name)
		}
		if _, err := io.Copy(tw, bytes.NewBuffer(bs)); err != nil {
			tw.Close()
			return sdk.WrapError(err, "Unable to copy pipeline buffer")
		}
	}

	// add generated applications to writer
	for _, a := range res.Applications {
		var app exportentities.Application
		if err := yaml.Unmarshal([]byte(a), &app); err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Cannot parse generated application"))
		}

		bs, err := exportentities.Marshal(app, exportentities.FormatYAML)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf("%s.app.yml", app.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			tw.Close()
			return sdk.WrapError(err, "Unable to write header for application %s", app.Name)
		}
		if _, err := io.Copy(tw, bytes.NewBuffer(bs)); err != nil {
			tw.Close()
			return sdk.WrapError(err, "Unable to copy application buffer")
		}
	}

	return nil
}