return {
  "doums/rg.nvim",
  cmd = { 'Rg', 'Rgf', 'Rgp', 'Rgfp' },
  config = function()
    require("rg").setup({
        excluded = {
            '.idea',
            'node_modules',
            '.git',
            'target',
            'package-lock.json',
            'Cargo.lock',
        },
    })
    end
}
