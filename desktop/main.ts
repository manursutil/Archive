// get the go binary - done
// start the go server - done 
// ping the go server - done 
// create the native webview window - done
// kill the server wehn closed - done

import { Webview } from "@webview/webview";
import { dirname, join } from "node:path";

const PORT = 8080;
const URL = `http://localhost:${PORT}`;

const bin = Deno.env.get("ARCHIVE_BIN") ?? join(dirname(Deno.execPath()), "archive");

const server = new Deno.Command(bin, {
  args: ["serve"],
  cwd: dirname(bin),
  stdout: "inherit",
  stderr: "inherit",
}).spawn();

const numAttempts = 50;

try {
  await waitForServer(numAttempts);

  const webview = new Webview();
  webview.title = "Archive";
  webview.navigate(URL);
  webview.run();
} finally {
  server.kill();
  await server.status;
}

async function waitForServer(attempts: number) {
  for (let i = 0; i < attempts; i++) {
    try {
      await fetch(`${URL}/items`).then((r) => r.body?.cancel());
      return;
    } catch {
      // 
      await new Promise((r) => setTimeout(r, 100));
    }
  }

  server.kill();
  throw new Error(`archive server did not start on ${URL}. Check port availability`);
  
}