local configs = require("lspconfig.configs")
if not configs.ty then
  configs.ty = {
    default_config = {
      cmd = { "ty", "server" },
      filetypes = { "python" },
      root_dir = require("lspconfig.util").root_pattern("pyproject.toml", "ty.toml", ".git"),
      single_file_support = true,
    },
  }
end

return {
  {
    "neovim/nvim-lspconfig",
    opts = {
      servers = {
        pyright = { enabled = false },
        basedpyright = { enabled = false },
        ty = { enabled = true },
        ruff = {
          on_attach = function(client)
            client.server_capabilities.hoverProvider = false
          end
        }
      },
    },
  },
}

