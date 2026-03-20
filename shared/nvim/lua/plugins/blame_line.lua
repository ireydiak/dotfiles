return {
  'braxtons12/blame_line.nvim',
  keys = {
    { '<leader>gb', '<cmd>BlameLineToggle<cr>', desc = 'Toggle blame line' },
  },
  config = function()
    require('blame_line').setup({
        show_in_visual = false,
        show_in_insert = false,
    })
    vim.cmd('BlameLineDisable')
  end
}

