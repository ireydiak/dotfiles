local ls = require("luasnip")
local s = ls.snippet
local t = ls.text_node
local i = ls.insert_node

return {
  s("ife", {
    t({
      "if err != nil {",
	  "",
    }),
    i(1, "return err"),
    t({
      "",
      "}",
    }),
  }),
}

