const one = require("./latest.json");
const two = require("./old-latest.json");

for (const o in one) {
  if (one[o] != two[o]) {
    console.log(o);
  }
}
