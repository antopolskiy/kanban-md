package agentname

// embeddedWords returns a curated list of short, evocative words suitable
// for agent names. Used as a fallback when the system dictionary is unavailable.
func embeddedWords() []string {
	return []string{
		"amber", "arch", "aspen", "atlas",
		"basin", "birch", "blade", "blaze", "bloom", "bolt", "bone", "brave", "brew", "brisk",
		"calm", "cape", "cedar", "char", "cider", "cliff", "cloud", "coal", "cold", "coral",
		"cove", "crane", "creek", "crisp", "crow", "crypt", "curl",
		"dale", "dart", "dawn", "deep", "delta", "dock", "dove", "drift", "dune", "dusk",
		"eagle", "echo", "edge", "elder", "ember", "fable", "falcon", "fawn", "fern", "field",
		"fire", "fjord", "flare", "flint", "flora", "flux", "foam", "forge", "fork", "fort",
		"frost", "gale", "gate", "ghost", "glade", "gleam", "glen", "glow", "gorge", "grain",
		"grove", "gust", "hail", "halo", "haven", "hawk", "hazel", "heath", "helm", "heron",
		"hill", "hive", "holly", "holt", "horn", "hull",
		"iron", "isle", "ivory", "jade", "keen", "kelp", "knoll",
		"lake", "larch", "lark", "lava", "leaf", "ledge", "light", "lily", "lime", "linen",
		"lodge", "lotus", "lunar", "lynx",
		"mace", "maple", "marsh", "mast", "mesa", "midge", "mill", "mink", "mint", "mist",
		"mold", "moss", "moth",
		"nectar", "north", "nova",
		"oaken", "opal", "orbit", "otter", "oxide",
		"palm", "path", "peak", "pearl", "pine", "plume", "pond", "port", "prism", "pulse",
		"quail", "quake", "quartz", "quick", "quiet", "quill",
		"rain", "rapid", "raven", "reef", "ridge", "rift", "river", "robin", "root", "rose",
		"rust", "sage", "salt", "sand", "shade", "shell", "shore", "silk", "slate", "sleet",
		"smoke", "snow", "solar", "south", "spark", "spire", "spoke", "spray", "spur", "staff",
		"star", "steam", "steel", "stem", "stone", "storm", "swift", "thorn", "thyme", "tide",
		"tiger", "timber", "torch", "trail", "trout", "twig", "vale", "valve", "vapor", "vault",
		"veil", "vine", "vivid", "void", "warp", "wave", "wheat", "wick", "wild", "willow",
		"wind", "wing", "wolf", "wren", "yarn", "zeal", "zinc", "zone",
	}
}
