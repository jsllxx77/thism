type ModulePreloadContext = {
  hostId: string
  hostType: "html" | "js"
}

export function filterModulePreloadDependencies(
  _url: string,
  deps: string[],
  context: ModulePreloadContext,
): string[] {
  if (context.hostType !== "html") {
    return deps
  }

  return deps.filter((dep) => !dep.includes("vendor-recharts"))
}
