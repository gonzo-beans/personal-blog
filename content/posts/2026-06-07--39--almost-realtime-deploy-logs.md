---
date: '2026-06-09T00:00:00-00:00'
title: 'Almost realtime deploy logs in my SaaS'
url: /blog/39
tags: ['saas-launch']
---

A totally unnecessary but cool change to my internal tools: realtime deploy logs in my SaaS.

Here's a preview of what was implemented:

<video controls class="">
    <source src="/videos/posts/39/wipa-analytics-pre-v1-deploy-demo_small.mp4">
</video>


{{< toc >}}

## Before the change. How logs are stored

Each tenant's host is provisioned and configured by a single internal service. This service has a Sqlite db with a table that tracks deployment history. Each row is a deployment and tracks the configuration for the deployment, the host, etc. The deployment logs are also stored in this table as a text column.

As the deploy progresses, we store the logs in a buffer. We flush this buffer stdout and to the Sqlite db once the deploy is finished. **This is a deliberate choice.** Concurrent deployments would be hard to follow if we incrementally flushed these logs to stdout. The internal service's logs would have deployments that interspersed with one another!

However, this also means that we can't query the database for an in-progress deployment. When a deployment takes more than a minute, it would be nice to see its progress.

## The storage approach

To idea here is simple. We already have a string buffer inside the service for each deploy. Each second, we could flush the buffer to sqlite. This works but each second the buffer grows and each subsequent write is larger than the last. This could be costly with long-running deployments that print a lot of logs. 

We might be pre-optimizing here, but we could flush to the DB only the bits that have been appended since the last flush. Using the following SQL

```sql
UPDATE deployments SET logs = logs || ? WHERE deployment_id = ?
```

We can wrap the logs buffer in a small struct that tracks the offset since the last flush.

```go
package deploy

import (
	"bytes"
	"sync"
)

type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	off int
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) FlushDelta() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	data := b.buf.Bytes()
	if b.off >= len(data) {
		return ""
	}
	delta := string(data[b.off:])
	b.off = len(data)
	return delta
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
```

If this code looks sloppy it's probably because it is. I picked up golang last year! I'm still learning.

## The presentation

This one is a bit trickier. In the internal service, I didn't opt for using a full web framework and use `htmx` instead.

With `htmx` we could use the `hx-trigger="every 1s"` approach. This works BUT we can't use the page template directly: This one renders the whole html doc we return to the client:  

```go-html-template
<!DOCTYPE html>
<html>
<head>
  <title>Deployment {{.Deployment.DeploymentID | printf "%.8s"}}</title>
  <link rel="stylesheet" href="/style.css">
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
</head>
<body class="deploy-detail">
  <h1>Deployment {{.Deployment.DeploymentID | printf "%.8s"}}</h1>
  <!-- Logs go here -->
</body>
</html>
```

We just have to break this information out into another template:

```go-html-template
<div id="deploy-content"
  {{if not .Deployment.DeployEnd}} 
    hx-get="/deployments/{{.Deployment.DeploymentID}}/content" 
    hx-trigger="every 1s" 
    hx-swap="outerHTML"
  {{end}}>
  <div class="meta">
    <!-- snip: this renders the bar with the deployment start, end, duration, etc. -->
  </div>
  <pre id="deploy-logs">{{.Deployment.Logs}}</pre>
</div>
```

This template will poll until the deployment finishes thanks to the use of `{{if not .Deployment.DeployEnd}}`.

This template is available at `GET /deployments/$DeploymentID/content`, so we can use that in the `hx-get` above and in the page's template:

```go-html-template
<!DOCTYPE html>
<html>
<head>
  <!--snip-->
</head>
<body class="deploy-detail">
  <h1>Deployment {{.Deployment.DeploymentID | printf "%.8s"}}</h1>
  
  <div id="deploy-content" 
    hx-get="/deployments/{{.Deployment.DeploymentID}}/content" 
    hx-trigger="load every 1s" 
    hx-swap="outerHTML"
  >
    <div class="meta"><span>Loading…</span></div>
  </div>
</body>
</html>
```

## Future improvements

We could clean this up a bit if we used [templ's fragments](https://templ.guide/syntax-and-usage/fragments/). These allow us to "tag" part of a template and render just that. In this case we could collapse these two templates into one file and reduce the duplication. Something to do another day! 

Also, this approach hijacks my scroll when I'm looking at logs _during_ an active deploy. I'll probably swap out htmx for something more integrated in the future. For now this works ok.

`logBuffer` is probably not optimal. I'm not familiar enough with go to understand the performance implications of the buffer so that's something to look into in the future.
