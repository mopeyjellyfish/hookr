import { defineConfig } from "vitepress";

const repoName = process.env.GITHUB_REPOSITORY?.split("/")[1];
const base =
	process.env.VITEPRESS_BASE ??
	(process.env.GITHUB_ACTIONS && repoName ? `/${repoName}/` : "/");

export default defineConfig({
	title: "Hookr",
	description: "Schema-defined WebAssembly plugin runtime for Go hosts and plugins.",
	base,
	cleanUrls: true,
	lastUpdated: true,
	themeConfig: {
		nav: [
			{ text: "Home", link: "/" },
			{ text: "Tutorials", link: "/tutorials/" },
			{ text: "How-To", link: "/how-to/" },
			{ text: "Reference", link: "/reference/" },
			{ text: "Explanation", link: "/explanation/" }
		],
		sidebar: [
			{
				text: "Tutorials",
				items: [
					{ text: "Overview", link: "/tutorials/" },
					{ text: "Build A URL Balancer Plugin", link: "/tutorials/urlbalancer" },
					{ text: "Build A Text Filter Plugin", link: "/tutorials/textfilter" }
				]
			},
			{
				text: "How-To",
				items: [
					{ text: "Overview", link: "/how-to/" },
					{ text: "Install Toolchain", link: "/how-to/install-toolchain" },
					{ text: "Generate Glue", link: "/how-to/generate-glue" },
					{ text: "Build Plugin", link: "/how-to/build-plugin" },
					{ text: "Open And Call Plugin", link: "/how-to/open-and-call-plugin" },
					{ text: "Inspect Contract", link: "/how-to/inspect-contract" },
					{ text: "Debug Plugin From CLI", link: "/how-to/debug-plugin-from-cli" },
					{ text: "Run Benchmarks", link: "/how-to/run-benchmarks" }
				]
			},
			{
				text: "Reference",
				items: [
					{ text: "Overview", link: "/reference/" },
					{ text: "CLI", link: "/reference/cli" },
					{ text: "ABI", link: "/reference/abi" },
					{ text: "Contract Model", link: "/reference/contracts" },
					{ text: "Generated Go API", link: "/reference/generated-go-api" },
					{ text: "Benchmark Snapshots", link: "/reference/benchmarks" }
				]
			},
			{
				text: "Explanation",
				items: [
					{ text: "Overview", link: "/explanation/" },
					{ text: "Architecture", link: "/explanation/architecture" },
					{ text: "Performance Model", link: "/explanation/performance-model" },
					{
						text: "Plugin Debugging And Development Tooling",
						link: "/explanation/plugin-debugging-tooling"
					},
					{ text: "Roadmap And Release", link: "/explanation/roadmap-and-release" }
				]
			},
			{
				text: "Benchmarks",
				items: [
					{
						text: "ABI Results 2026-02-19",
						link: "/benchmarks/abi-results-2026-02-19"
					},
					{
						text: "FlatBuffers Results 2026-03-09",
						link: "/benchmarks/flatbuffers-2026-03-09"
					}
				]
			}
		],
		socialLinks: [{ icon: "github", link: "https://github.com/mopeyjellyfish/hookr" }]
	}
});
