// netcfgd console. Plain DOM and EventSource — no framework, no build step.
"use strict";

const $ = (id) => document.getElementById(id);
const el = (tag, cls, text) => {
  const n = document.createElement(tag);
  if (cls) n.className = cls;
  if (text !== undefined) n.textContent = text;
  return n;
};

// ---------------------------------------------------------------- static form
// Show the static address fields only when static addressing is selected.
function wireStaticToggles() {
  const groups = new Map();
  document.querySelectorAll("input[data-toggle]").forEach((radio) => {
    const target = $(radio.dataset.toggle);
    if (!target) return;
    if (!groups.has(radio.name)) groups.set(radio.name, []);
    groups.get(radio.name).push(radio);
    radio.addEventListener("change", () => update(radio.name));
  });
  const update = (name) => {
    (groups.get(name) || []).forEach((r) => {
      const target = $(r.dataset.toggle);
      if (target) target.hidden = !(r.checked && r.value === "manual");
    });
  };
  groups.forEach((_, name) => update(name));
}

// ------------------------------------------------------------------- countdown
// The rollback guard reverts an interface change if nothing confirms it, so the
// operator needs to see how long they have left.
function wireCountdown() {
  const out = $("revert-countdown");
  if (!out) return;
  const tick = async () => {
    try {
      const r = await fetch("/api/status");
      const s = await r.json();
      if (!s.pending) {
        out.textContent = "";
        location.reload();
        return;
      }
      out.textContent = `reverting in ${s.pending.secondsLeft}s`;
    } catch (e) {
      out.textContent = "";
    }
  };
  tick();
  setInterval(tick, 2000);
}

// ----------------------------------------------------------------- probe render
const STAGE_LABEL = {
  dns: "DNS", tcp: "TCP", handshake: "Handshake", auth: "Auth",
  connect: "Connect", tls: "TLS", http: "HTTP",
};

function renderProbe(result, title, note) {
  const box = el("div", "probe " + (result.ok ? "ok" : "bad"));

  const h = el("h3");
  h.appendChild(el("span", null, title || result.target));
  h.appendChild(el("span", "note", result.ok ? `ok · ${result.millis} ms` : `failed at ${result.failed}`));
  box.appendChild(h);

  if (note) box.appendChild(el("div", "note", note));
  box.appendChild(el("div", "note", `via ${result.via}`));

  const table = el("table");
  (result.stages || []).forEach((s) => {
    const tr = el("tr", "stage " + (s.ok ? "ok" : s.skipped ? "skip" : "bad"));
    tr.appendChild(el("td", "name", STAGE_LABEL[s.name] || s.name));
    tr.appendChild(el("td", "mark", s.ok ? "pass" : s.skipped ? "—" : "fail"));
    tr.appendChild(el("td", "ms", s.ok ? `${s.millis} ms` : ""));

    const detail = el("td", "detail", s.error || s.detail || "");
    if (s.guidance) detail.appendChild(el("div", "guidance", s.guidance));
    tr.appendChild(detail);
    table.appendChild(tr);
  });
  box.appendChild(table);

  if (result.interception) box.appendChild(renderInterception(result.interception));
  return box;
}

// A re-encrypting proxy is worth spelling out: who is doing it, what it
// presented, and whether the CA needed to trust it is actually obtainable.
function renderInterception(i) {
  const box = el("div", "interception " + (i.trusted ? "ok" : "bad"));
  box.appendChild(el("h4", null, i.trusted
    ? "TLS is re-encrypted by a CA this device trusts"
    : "TLS is being re-encrypted by an untrusted CA"));

  const dl = el("dl", "compact");
  const add = (k, v) => { dl.appendChild(el("dt", null, k)); dl.appendChild(el("dd", null, v)); };
  add("Intercepted by", i.leafIssuer);
  box.appendChild(dl);

  const table = el("table");
  const head = el("tr");
  ["Certificate presented", "Issued by", "Type", "SHA-256"].forEach((h) => head.appendChild(el("td", "name", h)));
  table.appendChild(head);
  (i.chain || []).forEach((c) => {
    const tr = el("tr");
    tr.appendChild(el("td", null, c.subject));
    tr.appendChild(el("td", null, c.issuer));
    tr.appendChild(el("td", null, c.isCa ? (c.selfSigned ? "root CA" : "intermediate CA") : "leaf"));
    tr.appendChild(el("td", null, c.fingerprint));
    table.appendChild(tr);
  });
  box.appendChild(table);

  if (i.trusted) return box;

  if (i.caPresented && i.rootCa) {
    box.appendChild(el("p", "hint",
      "Check this fingerprint against what the network administrator says it should be, " +
      "then trust the CA in both places below."));
    const forConfig = el("details");
    forConfig.appendChild(el("summary", null, "balenaRootCA value for config.json"));
    forConfig.appendChild(el("pre", "console", i.rootCa.base64));
    box.appendChild(forConfig);
    const forConsole = el("details");
    forConsole.appendChild(el("summary", null, "PEM, for this console at /data/ca.pem"));
    forConsole.appendChild(el("pre", "console", i.rootCa.pem));
    box.appendChild(forConsole);
  } else {
    // The usual case. Say so plainly rather than offering the leaf, which
    // would look like trust material and be useless.
    box.appendChild(el("p", "hint",
      "The CA certificate itself was not sent — servers normally omit it, because clients " +
      "are expected to hold it already. Ask the network administrator for the root CA of \"" +
      i.leafIssuer + "\", then install it as balenaRootCA in config.json and at /data/ca.pem " +
      "for this console."));
  }
  return box;
}

function csrf() {
  const input = document.querySelector('input[name="csrf"]');
  return input ? input.value : "";
}

async function postJSON(url, body) {
  const r = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf() },
    body: JSON.stringify(body),
  });
  return r.json();
}

// ------------------------------------------------------------------ proxy test
function wireProxyTest() {
  const out = $("test-results");
  if (!out) return;

  const run = async (direct) => {
    out.replaceChildren(el("p", "hint", "Testing…"));
    const body = {
      direct,
      target: ($("test-target") || {}).value || "",
      type: (document.querySelector('select[name="type"]') || {}).value || "",
      ip: ($("proxy-ip") || {}).value || "",
      port: parseInt(($("proxy-port") || {}).value || "0", 10),
      login: ($("proxy-login") || {}).value || "",
      password: ($("proxy-password") || {}).value || "",
    };
    try {
      const res = await postJSON("/api/proxy/test", body);
      if (res.error) {
        out.replaceChildren(el("p", "err", res.error));
        return;
      }
      out.replaceChildren(renderProbe(res, direct ? "Direct (no proxy)" : "Through the proxy"));
    } catch (e) {
      out.replaceChildren(el("p", "err", String(e)));
    }
  };

  if ($("test-proxy")) $("test-proxy").addEventListener("click", () => run(false));
  if ($("test-direct")) $("test-direct").addEventListener("click", () => run(true));
}

// -------------------------------------------------------------- endpoint checks
function wireEndpoints() {
  const btn = $("run-endpoints");
  const out = $("endpoint-results");
  if (!btn || !out) return;

  btn.addEventListener("click", async () => {
    btn.disabled = true;
    out.replaceChildren(el("p", "hint", "Checking…"));
    try {
      const payload = await (await fetch("/api/endpoints")).json();
      const results = payload.results || [];
      out.replaceChildren();
      out.appendChild(el("p", "hint", "Tested " + payload.path));

      let group = "";
      results.forEach((r) => {
        if (r.endpoint.group !== group) {
          group = r.endpoint.group;
          out.appendChild(el("h3", null, group));
        }
        const title = r.endpoint.name + " · " + r.endpoint.addr +
          (r.endpoint.required ? "" : " (optional)");
        out.appendChild(renderProbe(r.result, title, r.endpoint.note));
      });

      const failed = results.filter((r) => !r.result.ok && r.endpoint.required);
      if (failed.length) {
        const summary = el("div", "banner error");
        summary.textContent = failed.length + " required endpoint(s) unreachable: " +
          failed.map((r) => r.endpoint.addr).join(", ");
        out.prepend(summary);
      } else {
        out.prepend(el("div", "banner ok", "Every required endpoint is reachable."));
      }
    } catch (e) {
      out.replaceChildren(el("p", "err", String(e)));
    } finally {
      btn.disabled = false;
    }
  });
}

// ----------------------------------------------------------------------- tunnel
function wireTunnel() {
  const btn = $("run-tunnel");
  const out = $("tunnel-results");
  if (!btn || !out) return;

  btn.addEventListener("click", async () => {
    btn.disabled = true;
    out.replaceChildren(el("p", "hint", "Checking…"));
    try {
      const payload = await (await fetch("/api/tunnel")).json();
      const t = payload.status;
      out.replaceChildren();

      const dl = el("dl", "compact");
      dl.appendChild(el("dt", null, "Edge tested"));
      dl.appendChild(el("dd", null, payload.path));
      const add = (k, v) => { dl.appendChild(el("dt", null, k)); dl.appendChild(el("dd", null, v)); };
      add("cloudflared registered", t.registered ? "yes" : "no");
      add("Edge connections", String(t.connections));
      add("TCP/7844 to the edge", payload.edge && payload.edge.ok ? "reachable" : "blocked");
      add("QUIC (UDP/7844)", t.quicViable ? "reachable" : "blocked");
      add("QUIC detail", t.quicDetail || "—");
      if (t.metricsError) add("Local metrics", t.metricsError);
      out.appendChild(dl);

      if (t.advice) out.appendChild(el("div", "banner warn", t.advice));
      if (payload.edge) out.appendChild(renderProbe(payload.edge, "Tunnel edge (region1)"));
    } catch (e) {
      out.replaceChildren(el("p", "err", String(e)));
    } finally {
      btn.disabled = false;
    }
  });
}

// -------------------------------------------------------------- streaming tools
// Runs an SSE endpoint, appending each event to a <pre> as it arrives.
function stream(url, out, btn, format) {
  out.textContent = "";
  btn.disabled = true;

  const src = new EventSource(url);
  const finish = () => { src.close(); btn.disabled = false; };

  Object.entries(format).forEach(([event, fn]) => {
    src.addEventListener(event, (e) => {
      out.textContent += fn(JSON.parse(e.data)) + "\n";
      out.scrollTop = out.scrollHeight;
    });
  });
  src.addEventListener("error", (e) => {
    // A payload means the server reported a problem; no payload means the
    // stream ended or the connection dropped.
    if (e.data) {
      try { out.textContent += JSON.parse(e.data).error + "\n"; } catch (_) {}
    }
    finish();
  });
  src.addEventListener("done", finish);
}

function wireTools() {
  if ($("run-ping")) {
    $("run-ping").addEventListener("click", () => {
      const host = encodeURIComponent($("ping-host").value);
      const count = encodeURIComponent($("ping-count").value);
      stream(`/api/ping?host=${host}&count=${count}`, $("ping-results"), $("run-ping"), {
        reply: (r) => r.timeout
          ? `seq ${r.seq}: timed out`
          : r.error
            ? `seq ${r.seq}: ${r.error}`
            : `seq ${r.seq}: ${r.rttMs.toFixed(1)} ms from ${r.from}`,
      });
    });
  }

  if ($("run-trace")) {
    $("run-trace").addEventListener("click", () => {
      const host = encodeURIComponent($("trace-host").value);
      const hops = encodeURIComponent($("trace-hops").value);
      stream(`/api/traceroute?host=${host}&maxHops=${hops}`, $("trace-results"), $("run-trace"), {
        hop: (h) => h.timeout
          ? `${String(h.ttl).padStart(2)}  *`
          : `${String(h.ttl).padStart(2)}  ${h.addr}${h.name ? " (" + h.name + ")" : ""}  ${h.rttMs.toFixed(1)} ms${h.final ? "  [destination]" : ""}`,
      });
    });
  }

  if ($("run-dns")) {
    $("run-dns").addEventListener("click", async () => {
      const out = $("dns-results");
      out.textContent = "Resolving…";
      try {
        const host = encodeURIComponent($("dns-host").value);
        const r = await (await fetch(`/api/dns?host=${host}`)).json();
        out.textContent = r.error
          ? `${r.host}: ${r.error}\nresolvers: ${(r.servers || []).join(", ") || "none"}`
          : `${r.host} -> ${(r.addrs || []).join(", ")}\nin ${r.millis} ms via ${(r.servers || []).join(", ") || "unknown resolver"}`;
      } catch (e) {
        out.textContent = String(e);
      }
    });
  }
}

// ------------------------------------------------------------------- wifi scan
function wireWifiScan() {
  const btn = $("scan-wifi");
  if (!btn) return;

  btn.addEventListener("click", async () => {
    const list = $("wifi-list");
    const out = $("wifi-results");
    btn.disabled = true;
    if (out) out.replaceChildren(el("p", "hint", "Scanning…"));
    try {
      const networks = await (await fetch("/api/wifi/scan")).json();
      if (networks.error) throw new Error(networks.error);

      if (list) {
        list.replaceChildren();
        networks.forEach((n) => {
          const o = document.createElement("option");
          o.value = n.ssid;
          list.appendChild(o);
        });
      }
      if (out) {
        const table = el("table");
        networks.forEach((n) => {
          const tr = el("tr");
          tr.appendChild(el("td", null, n.ssid));
          tr.appendChild(el("td", null, `${n.signalPct}%`));
          tr.appendChild(el("td", null, n.secured ? "secured" : "open"));
          table.appendChild(tr);
        });
        out.replaceChildren(table);
      }
    } catch (e) {
      if (out) out.replaceChildren(el("p", "err", String(e)));
    } finally {
      btn.disabled = false;
    }
  });
}

// ----------------------------------------------------------------- activity feed
function wireActivityStream() {
  const table = $("activity-table");
  if (!table) return;
  const tbody = table.querySelector("tbody");

  const src = new EventSource("/api/activity/stream");
  src.addEventListener("entry", (e) => {
    const entry = JSON.parse(e.data);
    const tr = el("tr", "lvl-" + entry.level);
    tr.appendChild(el("td", "ts", new Date(entry.at).toLocaleString()));
    tr.appendChild(el("td", null, entry.level));
    tr.appendChild(el("td", null, entry.source));
    tr.appendChild(el("td", null, entry.actor || "—"));

    const what = el("td", null, entry.message);
    if (entry.detail) what.appendChild(el("div", "muted", entry.detail));
    tr.appendChild(what);

    tbody.prepend(tr);
  });
}

wireStaticToggles();
wireCountdown();
wireProxyTest();
wireEndpoints();
wireTunnel();
wireTools();
wireWifiScan();
wireActivityStream();
