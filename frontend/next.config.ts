import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Menghasilkan server mandiri di .next/standalone untuk image Docker yang kecil.
  output: "standalone",
};

export default nextConfig;
