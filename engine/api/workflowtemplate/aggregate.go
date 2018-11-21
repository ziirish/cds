package workflowtemplate

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// AggregateAuditsOnWorkflowTemplate set audits for each workflow template.
func AggregateAuditsOnWorkflowTemplate(db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	as, err := GetAuditsByTemplateIDsAndEventTypes(db, sdk.WorkflowTemplatesToIDs(wts), []string{"WorkflowTemplateAdd", "WorkflowTemplateUpdate"})
	if err != nil {
		return err
	}

	m := map[int64][]sdk.AuditWorkflowTemplate{}
	for _, a := range as {
		if _, ok := m[a.WorkflowTemplateID]; !ok {
			m[a.WorkflowTemplateID] = []sdk.AuditWorkflowTemplate{}
		}
		m[a.WorkflowTemplateID] = append(m[a.WorkflowTemplateID], a)
	}

	// assume that audits are sorted by creation date with GetAudits
	for _, wt := range wts {
		if as, ok := m[wt.ID]; ok {
			wt.FirstAudit = &as[0]
			wt.LastAudit = &as[len(as)-1]
		}
	}

	return nil
}

// AggregateAuditsOnWorkflowTemplateInstance set audits for each workflow template instance.
func AggregateAuditsOnWorkflowTemplateInstance(db gorp.SqlExecutor, wtis ...*sdk.WorkflowTemplateInstance) error {
	as, err := GetInstanceAuditsByInstanceIDsAndEventTypes(db,
		sdk.WorkflowTemplateInstancesToIDs(wtis),
		[]string{"WorkflowTemplateInstanceAdd", "WorkflowTemplateInstanceUpdate"},
	)
	if err != nil {
		return err
	}

	m := map[int64][]sdk.AuditWorkflowTemplateInstance{}
	for _, a := range as {
		if _, ok := m[a.WorkflowTemplateInstanceID]; !ok {
			m[a.WorkflowTemplateInstanceID] = []sdk.AuditWorkflowTemplateInstance{}
		}
		m[a.WorkflowTemplateInstanceID] = append(m[a.WorkflowTemplateInstanceID], a)
	}

	// assume that audits are sorted by creation date with GetInstanceAudits
	for _, wti := range wtis {
		if as, ok := m[wti.ID]; ok {
			wti.FirstAudit = &as[0]
			wti.LastAudit = &as[len(as)-1]
		}
	}

	return nil
}

// AggregateTemplateOnWorkflow set template data for each workflow.
func AggregateTemplateOnWorkflow(db gorp.SqlExecutor, ws ...*sdk.Workflow) error {
	wtis, err := GetInstancesByWorkflowIDs(db, sdk.WorkflowToIDs(ws))
	if err != nil {
		return err
	}
	if len(wtis) == 0 {
		return nil
	}

	wts, err := GetAllByIDs(db, sdk.WorkflowTemplateInstancesToWorkflowTemplateIDs(wtis))
	if err != nil {
		return err
	}
	if len(wts) == 0 {
		return nil
	}

	mWorkflowTemplates := make(map[int64]sdk.WorkflowTemplate, len(wts))
	for _, wt := range wts {
		mWorkflowTemplates[wt.ID] = wt
	}

	mWorkflowTemplateInstances := make(map[int64]sdk.WorkflowTemplateInstance, len(wtis))
	for _, wti := range wtis {
		mWorkflowTemplateInstances[wti.WorkflowID] = wti
	}

	for _, w := range ws {
		if wti, ok := mWorkflowTemplateInstances[w.ID]; ok {
			if wt, ok := mWorkflowTemplates[wti.WorkflowTemplateID]; ok {
				w.Template = &wt
			}
		}
	}

	return nil
}