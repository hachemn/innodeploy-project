const http = require("http");

http.createServer((req, res) => {
  res.end("Hello from InnoDeploy 🚀");
}).listen(3000);