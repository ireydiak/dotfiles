local ls = require("luasnip")
local s = ls.snippet
local t = ls.text_node
local i = ls.insert_node

return {
  s("with", {
    t("with "),
    i(1, "expr"),
    t(" as "),
    i(2, "var"),
    t({
      ":",
      "\t",
    }),
    i(0, "VISUAL"),
  }),
}