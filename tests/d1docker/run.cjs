// Runs a Go WASM test binary with a real D1 binding as context.env.DB.
// usage: node run.cjs <wasm_exec_node.js> <test.wasm> [test flags]
// It is glue, not a test: every assertion lives in the Go test (conformance suites of webtyp/storage and webtyp/ddl).
const { Miniflare } = require("miniflare");

const [wasmExecNode, ...goArgs] = process.argv.slice(2);

async function main() {
  const mf = new Miniflare({
    modules: true,
    script: "export default { fetch() { return new Response('') } }",
    d1Databases: { DB: "conformance" },
  });
  const db = await mf.getD1Database("DB");
  globalThis.context = { env: { DB: db } };
  process.on("exit", () => { mf.dispose(); });
  process.argv = [process.argv[0], wasmExecNode, ...goArgs];
  require(wasmExecNode);
}

main().catch((err) => { console.error(err); process.exit(1); });
