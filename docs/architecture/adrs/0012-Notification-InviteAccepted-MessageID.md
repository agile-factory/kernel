# _ADR #0012:_ Notification P1-1 — `invite_accepted` handler + Message-ID deliverability fix

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

Lors du passage en P1-1, l'audit du module `platform/notification` a révélé deux défauts :

**Défaut 1 — Handler `auth.invite_accepted` était un stub no-op**
Le handler `handleInviteAcceptedStub` était enregistré depuis le début mais retournait `nil` sans envoyer d'email. Le commentaire indiquait "template pas encore livré". Les templates manquants ont été écrits (en + fr) et le vrai handler a été implémenté.

**Défaut 2 — `Message-ID` absent des emails SMTP**
`buildMIME` ne générait pas d'en-tête `Message-ID`. RFC 5322 recommande cet en-tête ; son absence est un signal spam pour Gmail, Outlook et les MTA modernes. De plus, `SMTPSender.Send` retournait toujours `""` comme `externalID`, rendant les lignes `notification_logs.external_id` inutilisables pour le suivi et le débogage.

**Défaut 3 — Parser de chemin de template cassé**
`renderer.go` utilisait `fmt.Sscanf` avec la syntaxe `%[^/]` (classe de caractères style C). Go's `fmt.Sscanf` ne supporte pas cette syntaxe — le module paniqueait au démarrage. Corrigé avec `strings.Split` + `strings.TrimSuffix`.

---

## **Decision**

### 1. Templates `invite_accepted` — en + fr

Ajout de 6 fichiers dans `templates/{en,fr}/invite_accepted.{subject,txt,html}.tmpl`. L'email notifie l'inviteur (OWNER ou ADMIN) quand son invité accepte l'invitation. Payload :

| Champ | Source | Usage |
|-------|--------|-------|
| `InviteeName` | outbox payload | Nom de l'invité |
| `InviteeEmail` | outbox payload | Email de l'invité |
| `InviterName` | outbox payload (fallback: préfixe email) | Destinataire |
| `Role` | outbox payload | Rôle accordé |
| `TenantName` | outbox payload (fallback: "your workspace") | Nom du workspace |
| `TeamURL` | `BaseURL + "/team"` | CTA vers la liste des membres |

Si `inviter_email` est vide (tenant créé sans invite), le handler retourne `nil` sans envoyer — comportement correct pour les inscriptions owner-direct.

### 2. `Message-ID` dans SMTP

`buildMIME(msg, msgID string)` accepte maintenant un `msgID` pré-généré (format `<uuid@host>`). `SMTPSender.Send` génère ce msgID, l'injecte dans le MIME, et le retourne comme `externalID`. Les lignes `notification_logs.external_id` sont maintenant exploitables pour le suivi.

En-têtes ajoutés : `Message-ID` + `Date` (RFC 5322).

### 3. Parser de chemin — `strings.Split` au lieu de `fmt.Sscanf`

```go
// Avant (cassé) — Go ne supporte pas %[^/]
fmt.Sscanf(path, "templates/%[^/]/%[^.].%[^.].tmpl", &locale, &name, &kind)

// Après (correct)
parts := strings.Split(path, "/")          // ["templates", "en", "email_verification.html.tmpl"]
filename := strings.TrimSuffix(parts[2], ".tmpl")  // "email_verification.html"
dot := strings.LastIndex(filename, ".")    // → name="email_verification", kind="html"
```

---

## **Alternatives Considered**

### Utiliser `path.Match` ou regexp pour le parser de template
- Plus expressif mais overkill pour un chemin structuré à 3 niveaux fixes.
- `strings.Split` est suffisant et lisible.
- **Verdict : rejeté**

### Générer un ID Resend/Postmark comme `externalID`
- Uniquement pertinent si un provider cloud est utilisé. `net/smtp` natif ne retourne pas d'ID provider.
- Le UUID v4 généré localement est suffisant pour le tracking interne.
- **Verdict : différé** — si un adapter Resend est ajouté (P1-4bis), il pourra retourner l'ID provider.

---

## **Consequences**

### **Pros**

- Le pipeline `auth.invite_accepted` → email est maintenant opérationnel end-to-end
- Les emails passent les filtres spam des MTA modernes (Message-ID + Date)
- `notification_logs.external_id` est exploitable pour le suivi et le replay
- Le renderer ne paniqie plus au démarrage
- 12 tests unitaires couvrent la notification (11 tests module + 5 sous-tests New validation)

### **Cons**

- Le `Message-ID` est généré localement (pas par le provider SMTP). Les bounces et DSN utiliseront cet ID, ce qui nécessite de parser les headers de bounce pour le réconcilier avec `notification_logs` — complexité différée à la gestion des bounces (post-P1).
