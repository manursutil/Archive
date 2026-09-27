// this allows for cmd+c, cmd+v on mac 

const objc = Deno.dlopen("/usr/lib/libobjc.A.dylib", {
  objc_getClass: { parameters: ["buffer"], result: "pointer" },
  sel_registerName: { parameters: ["buffer"], result: "pointer" },
  send0: {
    name: "objc_msgSend",
    parameters: ["pointer", "pointer"],
    result: "pointer",
  },
  send1: {
    name: "objc_msgSend",
    parameters: ["pointer", "pointer", "pointer"],
    result: "pointer",
  },
  sendStr: {
    name: "objc_msgSend",
    parameters: ["pointer", "pointer", "buffer"],
    result: "pointer",
  },
  send3: {
    name: "objc_msgSend",
    parameters: ["pointer", "pointer", "pointer", "pointer", "pointer"],
    result: "pointer",
  },
});

type Id = Deno.PointerValue;

const cstr = (s: string) => new TextEncoder().encode(s + "\0");
const cls = (name: string) => objc.symbols.objc_getClass(cstr(name));
const sel = (name: string) => objc.symbols.sel_registerName(cstr(name));
const nsString = (s: string) =>
  objc.symbols.sendStr(cls("NSString"), sel("stringWithUTF8String:"), cstr(s));

function send(obj: Id, selector: string, arg?: Id): Id {
  return arg === undefined
    ? objc.symbols.send0(obj, sel(selector))
    : objc.symbols.send1(obj, sel(selector), arg);
}

// [title, action, key]; an uppercase key implies ⇧
type Item = [string, string, string];

// a top-level menu bar entry holding a submenu of items
function submenu(title: string, items: Item[]): Id {
  const menu = send(
    send(cls("NSMenu"), "alloc"),
    "initWithTitle:",
    nsString(title),
  );

  for (const [label, action, key] of items) {
    const item = objc.symbols.send3(
      send(cls("NSMenuItem"), "alloc"),
      sel("initWithTitle:action:keyEquivalent:"),
      nsString(label),
      sel(action),
      nsString(key),
    );
    send(menu, "addItem:", item);
  }

  const entry = send(cls("NSMenuItem"), "new");
  send(entry, "setSubmenu:", menu);
  return entry;
}

// macOS only. Call after the Webview is created (it starts NSApplication),
// before run().
export function addEditMenu() {
  const app = send(cls("NSApplication"), "sharedApplication");
  let bar = send(app, "mainMenu");

  if (!bar) {
    bar = send(cls("NSMenu"), "new");
    // the first entry is always the app menu, whatever its title. Quit closes
    // the window rather than terminate:, which would exit() before main.ts
    // stops the server.
    send(
      bar,
      "addItem:",
      submenu("Archive", [["Quit Archive", "performClose:", "q"]]),
    );
    send(app, "setMainMenu:", bar);
  }

  // nil-target actions go to the focused view, i.e. the webview
  send(
    bar,
    "addItem:",
    submenu("Edit", [
      ["Undo", "undo:", "z"],
      ["Redo", "redo:", "Z"],
      ["Cut", "cut:", "x"],
      ["Copy", "copy:", "c"],
      ["Paste", "paste:", "v"],
      ["Select All", "selectAll:", "a"],
    ]),
  );
}
