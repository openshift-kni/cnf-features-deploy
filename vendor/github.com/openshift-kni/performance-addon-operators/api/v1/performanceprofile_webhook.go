package v1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	webhookPort     = 4343
	webhookCertDir  = "/apiserver.local.config/certificates"
	webhookCertName = "apiserver.crt"
	webhookKeyName  = "apiserver.key"
)

// SetupWebhookWithManager enables Webhooks - needed for version conversion
func (r *PerformanceProfile) SetupWebhookWithManager(mgr ctrl.Manager) error {
	bldr := ctrl.NewWebhookManagedBy(mgr).
		For(r)

	// Specify OLM CA Info
	srv := mgr.GetWebhookServer()
	srv.CertDir = webhookCertDir
	srv.CertName = webhookCertName
	srv.KeyName = webhookKeyName
	srv.Port = webhookPort

	return bldr.Complete()
}
