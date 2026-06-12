---
name: graph-api
description: Answer questions about Microsoft Graph API, Entra ID OAuth, mail sync, token refresh, and thread grouping for this project. Accepts a topic and gives a targeted, actionable answer using the reference below.
when_to_use: Use when the user asks about Graph API endpoints, OAuth scopes, paging, token refresh, conversationId grouping, Phase 2 In-Reply-To tree, or how mail sync is implemented in this project.
argument-hint: "[topic: oauth | mail-query | paging | thread-grouping | token-refresh | phase2]"
allowed-tools: Read, Grep, WebFetch
---

If the user provided a topic via `$ARGUMENTS`, focus your answer on that section.
If no argument was given, ask: "Which topic? `oauth` · `mail-query` · `paging` · `thread-grouping` · `token-refresh` · `phase2`"

Always answer with concrete code or endpoint examples — not just theory.
If the reference below is insufficient, WebFetch the official Graph API docs before answering.

---

## oauth — Scopes to register in Entra ID

```
openid profile email offline_access Mail.Read
```
`offline_access` is required to get a refresh token. Register these in the Entra ID app registration under **API permissions**.

---

## mail-query — Pull mail from Inbox

```
GET https://graph.microsoft.com/v1.0/me/mailFolders/Inbox/messages
  ?$select=id,conversationId,subject,from,toRecipients,ccRecipients,receivedDateTime,bodyPreview,body,internetMessageId
  &$filter=receivedDateTime ge 2024-01-01T00:00:00Z
  &$orderby=receivedDateTime desc
  &$top=50
```

Use the user's decrypted `access_token` from `user_tokens` as `Authorization: Bearer <token>`.

---

## paging — Follow @odata.nextLink

Response includes `@odata.nextLink` when more pages exist. Loop until the field is absent:

```go
for {
    resp, err := callGraph(url, token)
    messages = append(messages, resp.Value...)
    if resp.NextLink == "" {
        break
    }
    url = resp.NextLink
}
```

---

## thread-grouping — Phase 1 (current)

Group by `conversationId`, sort each group `receivedDateTime asc`:

```go
threads := groupByConversationID(messages)
for _, thread := range threads {
    sort.Slice(thread, func(i, j int) bool {
        return thread[i].ReceivedDateTime.Before(thread[j].ReceivedDateTime)
    })
}
```

---

## token-refresh — When access token expires

Call refresh on 401 from Graph API, then retry the original request:

```
POST https://login.microsoftonline.com/{tenant_id}/oauth2/v2.0/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=<decrypted_rt>
&client_id=<AZURE_CLIENT_ID>
&client_secret=<AZURE_CLIENT_SECRET>
```

On success → encrypt new tokens → update `user_tokens` in DB.
On failure → set `needs_reauth=true` → return 401 `{"error":"reauth_required"}`.

---

## phase2 — Thread branching via In-Reply-To (implement only after user feedback)

Add `internetMessageHeaders` to `$select`:

```
?$select=...,internetMessageHeaders
```

Parse `In-Reply-To` header → build parent-child map → flatten depth-first before summarizing.
**Do not implement until users report that Phase 1 sort is insufficient.**
