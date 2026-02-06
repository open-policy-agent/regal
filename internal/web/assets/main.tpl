<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8"/>
  <link rel="stylesheet" href="/assets/missing.min.css">
  <script src="/assets/htmx-1.8.4.min.js"></script>
</head>
<body>
<main class="crowded">
  <details class="info">
    <summary>Policy</summary>
      <pre><code>{{ .Code }}</code></pre>
  </details>
  <form>
    <div class="f-row">
      <input hx-get="?tmpl=output"
        hx-target="#output"
        hx-trigger="change"
        hx-include="form"
        type="checkbox"
        name="hide_identical"
        id="hide_identical">
      <label for="hide_identical">Hide stages without effect on code</label>
      <input hx-get="?tmpl=output"
        hx-target="#output"
        hx-trigger="change"
        hx-include="form"
        type="checkbox"
        name="print"
        id="print">
      <label for="print">Enable print</label>
      <input hx-get="?tmpl=output"
        hx-target="#output"
        hx-trigger="change"
        hx-include="form"
        type="checkbox"
        name="format"
        id="format">
      <label for="format">Format stages</label>
    </div>
  </form>
  <section id="output">
    {{ block "output" . }}
    {{ range .Result }}
    <details class="{{ .Class }}" {{ if .Show }}open{{ end }}>
      <summary>{{ .Stage }}</summary>
      <pre><code>{{ .Output }}</code></pre>
    </details>
    {{ end }}
    {{ end }}
  </section>
</main>
</body>
</html>
