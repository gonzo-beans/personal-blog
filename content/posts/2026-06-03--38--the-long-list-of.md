---
date: '2026-06-08T00:00:00-00:00'
title: 'The long list of bad decisions I made for my new SaaS'
url: /blog/38
tags: ['saas-launch']
# draft: true
---

I'm about to launch a new analytics SaaS. Easily the biggest project I've ever taken on myself. I don't usually finish ANY projects so this is a big life goal for me.

I have humble goals for myself: get two subscribers by the end of the year. There's six months left to the year so if I can average bringing in a third-of-a-person each month I got that goal in the bag. If you know any halflings hit me up.

{{< toc >}}

## Going against conventional wisdom. Look ma, no containers

The weird thing about my architecture design is that I opted to segregate tenants into their own VMs instead of their own containers. I'm probably leaving money on the table but I'd rather do it this way since it simplifies security issues relating to tenant isolation.

That said, it's not the only weird part of the architecture. I'm using `Svelte` instead of `React` and I'm rendering server templates as well. It's _almost_ a homemade server-side rendering setup. The first page waits for the js bundle to load and render, but there's no client side fetching of APIs on first load. The backend server provides it as props instead. It's a taped together SSR experience that sounds way worse than it feels to use.

## The long list. Did I paint myself into a corner?

So far just with those two decisions I'm going wayyy against conventional wisdom. I would have saved way more time if I took off the shelve tech and integrated it together.

But weirdly enough I don't regret my decision at all.

> I really hope I'm not sprinting head first into a pole that's staying perfectly in my blindspot.

To illustrate my point, let's make a giant list of possibly _catastrophic_ decisions.

- Wrote a crude continous deployment process into [Hetzner VMs](https://www.hetzner.com/cloud/) instead of using [Fly.io](https://fly.io)/[Railway](https://railway.com/)/[AWS Lightsail](https://aws.amazon.com/lightsail/)/[exe.dev](https://exe.dev).
- Deploying services as compiled golang binaries over SFTP to bare VMs instead of using something like [Docker Swarm](https://docs.docker.com/engine/swarm/)/[ansible](https://docs.ansible.com/projects/ansible/latest/index.html)/[pyinfra](https://github.com/pyinfra-dev/pyinfra) .
- Wrote my own load generator instead of using [wrk](https://github.com/wg/wrk)/[vegeta](https://github.com/tsenart/vegeta)/[otelgen](https://github.com/krzko/otelgen).
- Using [goyek](https://github.com/goyek/goyek) as a build system to replace my [just](https://github.com/casey/just) scripts. I could have used [make](https://www.gnu.org/software/make/) here.
- Wrote custom pagination in SQL instead of using an ORM or query builder.
- Web analytics over websockets instead of HTTP events. I can't use [navigator.sendBeacon()](https://developer.mozilla.org/en-US/docs/Web/API/Navigator/sendBeacon) here!
- Using the otel collector libraries instead of the [otelcol utility](https://opentelemetry.io/docs/collector/).
- Generating browser client certs for mutual TLS without using [openssl](https://www.openssl.org/).
- Using [Litestream](https://litestream.io/) to continuously back up SQLite to S3 instead of using a managed database. **This one is the one I'm most worried about.**
- Homemade SSR with Go templates + [Svelte](https://svelte.dev/) mount instead of [Next.js](https://nextjs.org/). Mentioned before. Also have I mentioned I hate using React?
- Using [Incus (LXC)](https://linuxcontainers.org/incus/introduction/) for the dev environment instead of Docker Compose.
- Encrypting production secrets with age+Scrypt passphrase instead of HashiCorp Vault or SOPS. This will be replaced eventually!
- No frontend tests. I got tired of fighting with them. They will eventually come back but after I hire someone full time.
- Embedding the IP-to-country database into the analytics service binary. Avoids consulting another API on the ingestion hot path.
- Using go's crypto/ssh library instead of `ssh-keyscan` when a new VM comes online.

## The zen of using my own tools

I already mentioned I don't regret my decision. A big contributor to that is that it's just so frictionless to work on the project. That's to be expected. I've setup everything exactly how I like it. It feels like walking into an organized workshop with tools you've never seen before haha.

I use the incus container to test the deployment steps. The logic in these steps are reused during production deployments. I only have to swap out the container provisioning for vm provisioning.

Each task is clearly labeled and timed!

<script src="https://asciinema.org/a/qZSL7S8RfjKEkSdV.js" id="asciicast-qZSL7S8RfjKEkSdV" async="true"></script>

With the incus container running the database, we can run the integration tests against it. These spin up the go services within the process that executes the integration tests using non-overlapping ports and they run in parallel. So despite the fact that there's **98 integration tests currently**, they finish quite quickly.

<script src="https://asciinema.org/a/N9W416GvVOSPgDKy.js" id="asciicast-N9W416GvVOSPgDKy" async="true"></script>

Granted, the majority of that speed comes from the fact we don't clean the database between each test. Each one is logically isolated from the other.

Still, the use of [goyek](https://github.com/goyek/goyek) here allows us to really leverage the excellent level of concurrency that golang can offer. Check out this lint task. It lints **19 backend modules and 2 frontend modules**. It lints them all concurrently without mixing outputs.

<script src="https://asciinema.org/a/YdY2bifqd2YlYTia.js" id="asciicast-YdY2bifqd2YlYTia" async="true"></script>

Notice the whole task doesn't take longer than any one individual part. It's beyond cool!

Writing the load generator was also one of the most dubious decisions as far as wasting time vs getting me closer to a finished marketable product. But it turned out to save me a TON of time. I took deep inspiration from the ideas behind the excellent python load testing tool [Locust](https://docs.locust.io/en/stable/what-is-locust.html).

I now have a declarative approach to defining load tests with virtual users that do multiple actions with rich meaning and coherence. There's also traffic shaping (so there's less users at midnight and peak users during the day) and a backfill mode that respects both the previous properties. It's also capable of high-throughput, again thanks to golang's excellent concurrency.

<script src="https://asciinema.org/a/hQN6sAfdcwxUVByK.js" id="asciicast-hQN6sAfdcwxUVByK" async="true"></script>

In 2 seconds we have some very rich test data to play with.

![Screenshot of the analytics page after running otelgen](/img/posts/38/wip-analytics-post-otelgen-vm-analytics-screenshot.png)

Also in the logs
![Screenshot of the logs page after running otelgen](/img/posts/38/wip-analytics-post-otelgen-vm-logs-screenshot.png)

The last weird trick is the developing experience in the SaaS client's portal. The two screenshots above come from that portal.

There's two modes to it. The first one is the **deployed mode**. This one uses the bundled version of the frontend. These assets are fingerprinted and cached "forever" by the browser so navigating between pages feels snappy:

<video controls class="">
    <source src="/videos/posts/38/wip-analytics-pre-v1-vm-demo_small.mp4">
</video>

I'm not breaking any speed records but keep in mind I'm using a remote server in another continent. 

> The video above was connection ping of 130ms to 150ms. Pretty snappy despite that handicap!

The second mode for the client's portal is the **"dev" mode**. This is the more traditional mode where the page auto-reloads when you change the source files. This one was tricky to setup since the golang server needs to restart on go changes and the frontend code needs to reload on js/css changes. The latter is handled by vite. The former by [air](https://github.com/air-verse/air). I also had to wire up a proxy path in dev mode where air forwards requests to vite. It seems fragile but it works surprisingly well.

I use dev mode when I want to quickly iterate on frontend changes without going through a whole container rebuild/deploy cycle. I just simply change the URL from localhost:8080 to localhost:8081.

<video controls class="">
    <source src="/videos/posts/38/wip-analytics-pre-v1-dev-demo_small.mp4">
</video>

The initial page load takes around 4.5 seconds, but the updates are instant (edits are off-screen so just trust me ok?). For that reason this mode is really just reserved for tweaking interface details.

## What's next for the product?

I've deployed some instances but I haven't officially launched yet. I'm going to ask some friends if they'd like to test out the service ✨ for free ✨. Maybe I can get them hooked on it and turn them into clients. If one of them converts I'll be super happy.

If you'd like to try it out reach out to me via the email in my [github profile](https://github.com/gonzo-beans).

The next couple of months will be the tedious dance of user testing, tweaks, and marketing. The nerd in me dreads the marketing but I've matured enough to understand how the world works. I'm no longer illusioned with the idea I'll get throngs of people begging to give me money just because I built something I think is cool. A boy can dream though.
