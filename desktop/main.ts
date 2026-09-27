// get the go binary - done
// start the go server - done
// ping the go server - done
// create the native webview window - done
// kill the server wehn closed - done

import { Webview } from "@webview/webview";
import { dirname, join } from "node:path";

const PORT = 8080;
const URL = `http://localhost:${PORT}`;

const bin = Deno.env.get("ARCHIVE_BIN") ??
  join(dirname(Deno.execPath()), "archive");

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
  if (Deno.build.os === "darwin") (await import("./menu.ts")).addEditMenu();
  webview.bind("openExternal", openExternal);
  webview.bind("pickFile", pickFile);
  webview.navigate(URL);
  webview.run();
} finally {
  server.kill();
  await server.status;
}

// use system browser to open saved html.
function openExternal(url: string) {
  if (!/^https?:\/\//.test(url)) return;

  const os = Deno.build.os;
  const cmd = os === "darwin"
    ? "open"
    : os === "windows"
    ? "explorer"
    : "xdg-open";
  new Deno.Command(cmd, { args: [url] }).spawn();
}

// native file picker
function pickFile(): string {
  const [cmd, ...args] = Deno.build.os === "darwin"
    ? ["osascript", "-e", 'POSIX path of (choose file of type {"txt", "md"})']
    : ["zenity", "--file-selection", "--file-filter=*.txt *.md"];

  const out = new Deno.Command(cmd, { args }).outputSync();
  return out.success ? new TextDecoder().decode(out.stdout).trim() : "";
}

async function waitForServer(attempts: number) {
  for (let i = 0; i < attempts; i++) {
    try {
      await fetch(`${URL}/items`).then((r) => r.body?.cancel());
      return;
    } catch {
      await new Promise((r) => setTimeout(r, 100));
    }
  }

  server.kill();
  throw new Error(
    `archive server did not start on ${URL}. Check port availability`,
  );
}
