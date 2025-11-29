const express = require('express');
const path = require('path');

const app = express();

app.use(express.json());
app.use(express.static(__dirname));

app.listen(5173, () => {
  console.log("Test frontend running at http://localhost:5173");
});
