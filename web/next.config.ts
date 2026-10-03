import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // standalone — локальный Dockerfile.web; export — статика в nginx (Dockerfile.nginx).
  output: process.env.NEXT_OUTPUT === "export" ? "export" : "standalone",
};

export default nextConfig;
