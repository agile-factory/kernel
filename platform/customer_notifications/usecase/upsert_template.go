package usecase

import (
	"context"
	htmltmpl "html/template"
	texttmpl "text/template"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
)

// UpsertCustomerTemplateUseCase creates or replaces a per-tenant template override.
type UpsertCustomerTemplateUseCase struct {
	repo    domain.CustomerTemplateRepository
	checker permission.Checker
}

// NewUpsertCustomerTemplateUseCase wires the use case.
func NewUpsertCustomerTemplateUseCase(repo domain.CustomerTemplateRepository, checker permission.Checker) *UpsertCustomerTemplateUseCase {
	return &UpsertCustomerTemplateUseCase{repo: repo, checker: checker}
}

// UpsertCustomerTemplateInput holds the template override parameters.
type UpsertCustomerTemplateInput struct {
	Key         string
	Channel     domain.Channel
	Locale      string
	SubjectTmpl string
	BodyTmpl    string
}

// Execute upserts the template after validating permissions and syntax.
func (uc *UpsertCustomerTemplateUseCase) Execute(ctx context.Context, tenantID string, in UpsertCustomerTemplateInput) error {
	if err := uc.checker.Require(ctx, permission.CustomerTemplateWrite); err != nil {
		return err
	}

	if !domain.SupportedTemplateKeys[in.Key] {
		return domain.ErrUnsupportedTemplateKey
	}
	if in.Channel != domain.ChannelEmail && in.Channel != domain.ChannelSMS {
		return kerrors.BadRequest("channel must be EMAIL or SMS")
	}
	if in.Locale == "" {
		return kerrors.BadRequest("locale is required")
	}
	if in.BodyTmpl == "" {
		return kerrors.BadRequest("body_tmpl is required")
	}

	// Parse templates at upsert time (Lucas amendment: catch syntax errors early).
	// Use html/template for EMAIL (XSS-safe), text/template for SMS.
	if in.Channel == domain.ChannelEmail {
		if _, err := htmltmpl.New("body").Parse(in.BodyTmpl); err != nil {
			return domain.ErrTemplateInvalid
		}
		if in.SubjectTmpl != "" {
			if _, err := htmltmpl.New("subject").Parse(in.SubjectTmpl); err != nil {
				return domain.ErrTemplateInvalid
			}
		}
	} else {
		if _, err := texttmpl.New("body").Parse(in.BodyTmpl); err != nil {
			return domain.ErrTemplateInvalid
		}
		if in.SubjectTmpl != "" {
			if _, err := texttmpl.New("subject").Parse(in.SubjectTmpl); err != nil {
				return domain.ErrTemplateInvalid
			}
		}
	}

	return uc.repo.Upsert(ctx, &domain.CustomerTemplate{
		TenantID:    tenantID,
		Key:         in.Key,
		Channel:     in.Channel,
		Locale:      in.Locale,
		SubjectTmpl: in.SubjectTmpl,
		BodyTmpl:    in.BodyTmpl,
	})
}

// DeleteCustomerTemplateUseCase removes a per-tenant template override (reverts to default).
type DeleteCustomerTemplateUseCase struct {
	repo    domain.CustomerTemplateRepository
	checker permission.Checker
}

// NewDeleteCustomerTemplateUseCase wires the use case.
func NewDeleteCustomerTemplateUseCase(repo domain.CustomerTemplateRepository, checker permission.Checker) *DeleteCustomerTemplateUseCase {
	return &DeleteCustomerTemplateUseCase{repo: repo, checker: checker}
}

// Execute removes the template override.
func (uc *DeleteCustomerTemplateUseCase) Execute(ctx context.Context, tenantID, key string, ch domain.Channel, locale string) error {
	if err := uc.checker.Require(ctx, permission.CustomerTemplateWrite); err != nil {
		return err
	}
	return uc.repo.Delete(ctx, tenantID, key, ch, locale)
}
