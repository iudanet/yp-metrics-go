package server

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Метрики</title>
</head>
<body>
    <h1>Метрики Counters</h1>
    <ul>
    {{range $key, $value := .Counters}}
        <li>{{$key}}: {{$value}}</li>
    {{end}}
    </ul>
    <h1>Метрики Gauges</h1>
    <ul>
    {{range $key, $value := .Gauges}}
        <li>{{$key}}: {{printf "%4.3f" $value}}</li>
    {{end}}
    </ul>
</body>
</html>`
