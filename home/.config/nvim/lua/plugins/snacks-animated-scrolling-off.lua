return {
	"folke/snacks.nvim",
	opts = {
        picker = {
            sources = {
                files = {
                    hidden = false, -- Display hidden files
                    ignored = true, -- Display gitignored files
                },
            },
        },
		scroll = {
			enabled = false, -- Disable scrolling animations
		},
	},
}

