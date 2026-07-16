package htmlreport

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>ComplexityRadar Report — {{.Project.Name}}</title>
<style>
  :root {
    --bg: #f5f6f8; --card: #ffffff; --ink: #1c2530; --muted: #6b7784;
    --line: #e3e7ec; --accent: #2b5dc7;
    --good-bg: #e6f4ea; --good-fg: #1e7a34;
    --warn-bg: #fff3e0; --warn-fg: #a15c00;
    --bad-bg: #fdecea; --bad-fg: #c0392b;
    --empty-bg: #eef0f2; --empty-fg: #7a828b;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; background: var(--bg); color: var(--ink);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    line-height: 1.5; font-size: 15px;
  }
  .wrap { max-width: 960px; margin: 0 auto; padding: 32px 20px 64px; }
  h1 { font-size: 1.6rem; margin: 0 0 4px; }
  h2 { font-size: 1.25rem; margin: 40px 0 16px; padding-top: 16px; border-top: 2px solid var(--line); }
  h3 { font-size: 1rem; margin: 24px 0 8px; color: var(--muted); text-transform: uppercase; letter-spacing: .04em; }
  .sub { color: var(--muted); margin: 0 0 4px; }
  .card {
    background: var(--card); border: 1px solid var(--line); border-radius: 12px;
    padding: 24px; margin-bottom: 20px; box-shadow: 0 1px 2px rgba(0,0,0,.04);
  }
  .hero { display: flex; align-items: center; gap: 28px; flex-wrap: wrap; }
  .hero .meta { flex: 1 1 240px; }
  .score-box { text-align: center; min-width: 150px; padding: 16px 20px; border-radius: 12px; }
  .score-num { font-size: 3.2rem; font-weight: 700; line-height: 1; }
  .score-max { font-size: 1rem; color: var(--muted); }
  .grade { display: inline-block; margin-top: 8px; font-size: 1.1rem; font-weight: 700;
    padding: 2px 12px; border-radius: 999px; }
  .rollup-tag { display: inline-block; font-size: .8rem; font-weight: 600; color: var(--accent);
    background: #eaf0fb; border-radius: 999px; padding: 2px 10px; margin-left: 8px; vertical-align: middle; }
  .legend { color: var(--muted); font-size: .85rem; margin-top: 12px; }

  .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 14px; }
  .tile { border: 1px solid var(--line); border-radius: 10px; padding: 16px; }
  .tile .dim { font-weight: 600; margin-bottom: 6px; }
  .tile .tscore { font-size: 2rem; font-weight: 700; line-height: 1; }
  .tile .tmeta { color: var(--muted); font-size: .85rem; margin-top: 6px; }
  .tile .badge { float: right; font-weight: 700; padding: 1px 10px; border-radius: 999px; font-size: .85rem; }

  table { width: 100%; border-collapse: collapse; margin-top: 8px; }
  th, td { text-align: left; padding: 9px 10px; border-bottom: 1px solid var(--line); font-size: .92rem; }
  th { color: var(--muted); font-weight: 600; font-size: .78rem; text-transform: uppercase; letter-spacing: .03em; }
  td.num, th.num { text-align: right; font-variant-numeric: tabular-nums; }
  .metric-name { font-weight: 600; }
  .method { color: var(--muted); font-size: .8rem; margin-top: 2px; }
  .pill { display: inline-block; font-weight: 700; padding: 1px 9px; border-radius: 999px; min-width: 44px; text-align: center; }

  .good { background: var(--good-bg); color: var(--good-fg); }
  .warn { background: var(--warn-bg); color: var(--warn-fg); }
  .bad  { background: var(--bad-bg);  color: var(--bad-fg); }
  .empty{ background: var(--empty-bg);color: var(--empty-fg); }

  .errors { border-left: 4px solid var(--bad-fg); background: var(--bad-bg); padding: 12px 16px; border-radius: 6px; }
  .errors ul { margin: 6px 0 0; padding-left: 18px; }
  footer { color: var(--muted); font-size: .82rem; text-align: center; margin-top: 40px; }

  @media (max-width: 520px) { .score-box { width: 100%; } }
  @media print {
    body { background: #fff; }
    .card { box-shadow: none; break-inside: avoid; }
    .tile, tr { break-inside: avoid; }
    h2 { break-before: page; }
  }
</style>
</head>
<body>
<div class="wrap">
  <h1>ComplexityRadar Report</h1>
  <p class="sub">Repository &amp; project health across security, delivery, infrastructure and code.</p>

  {{template "report" .Project}}

  {{if .Repos}}
    <h2>Per-repository detail</h2>
    {{range .Repos}}{{template "report" .}}{{end}}
  {{end}}

  <footer>Generated {{.Generated}} · Scores 0–100, higher is healthier · A ≥90 · B ≥75 · C ≥60 · D ≥40 · F &lt;40</footer>
</div>
</body>
</html>

{{define "report"}}
<section class="card">
  <div class="hero">
    <div class="meta">
      <h1 style="font-size:1.3rem;margin:0 0 4px">{{.Name}}{{if .Aggregate}}<span class="rollup-tag">project rollup</span>{{end}}</h1>
      {{if .Description}}<p class="sub">{{.Description}}</p>{{end}}
      <p class="sub">Collected {{.Collected}}</p>
    </div>
    <div class="score-box {{.Band}}">
      <div><span class="score-num">{{printf "%.1f" .Overall}}</span><span class="score-max"> / 100</span></div>
      <div class="grade {{.Band}}">Grade {{.Grade}}</div>
    </div>
  </div>

  {{if .Dimensions}}
  <div class="cards" style="margin-top:20px">
    {{range .Dimensions}}
    <div class="tile">
      <div class="dim">{{.Name}}<span class="badge {{.Band}}">{{.Grade}}</span></div>
      {{if .MetricCount}}
        <div class="tscore {{.Band}}" style="background:none;padding:0">{{printf "%.1f" .Score}}</div>
      {{else}}
        <div class="tscore empty" style="background:none;padding:0">—</div>
      {{end}}
      <div class="tmeta">Weight {{printf "%.0f" .Weight}}% · {{.MetricCount}} metric{{if ne .MetricCount 1}}s{{end}}{{if .Breakdown}}<br>{{.Breakdown}}{{end}}</div>
    </div>
    {{end}}
  </div>
  {{end}}

  {{range .Groups}}
  <h3>{{.Dimension}}</h3>
  <table>
    <thead><tr><th>Metric</th><th class="num">Raw</th><th>Unit</th><th class="num">Score</th></tr></thead>
    <tbody>
    {{range .Metrics}}
      <tr>
        <td>
          <div class="metric-name" title="{{.Tooltip}}">{{.Name}}</div>
          {{if .ScoreDef}}<div class="method">{{.ScoreDef}}</div>{{end}}
        </td>
        <td class="num">{{.Raw}}</td>
        <td>{{.Unit}}</td>
        <td class="num"><span class="pill {{.Band}}">{{.Score}}</span></td>
      </tr>
    {{end}}
    </tbody>
  </table>
  {{end}}

  {{if .Errors}}
  <div class="errors" style="margin-top:20px">
    <strong>Errors</strong>
    <ul>{{range .Errors}}<li>{{.}}</li>{{end}}</ul>
  </div>
  {{end}}
</section>
{{end}}
`
