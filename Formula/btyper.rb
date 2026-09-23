class Btyper < Formula
  desc "Adaptive touch-typing practice in the terminal"
  homepage "https://github.com/Ada-lave/btyper"
  version "1.0.1"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.1/btyper-darwin-amd64", using: :nounzip
      sha256 "97295c0a5429653555c272fee00b1bef2232b3613142c013181482055dfbd57b"
    end
    on_arm do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.1/btyper-darwin-arm64", using: :nounzip
      sha256 "19dc2ac8c1ba14f5b9c8a15da418116569141497933552cd0cf65511469de1ac"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.1/btyper-linux-amd64", using: :nounzip
      sha256 "9573a3fcab3bd8a043fa3e9f3f41b8e9b7308bebfe7d0a3610fe552138ab81f7"
    end
    on_arm do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.1/btyper-linux-arm64", using: :nounzip
      sha256 "974be9cf081d126b4aa8ccba4e2e9dae44ebd14da9955c9202a6c71420f4d2a4"
    end
  end

  def install
    binary = Dir["btyper-*"].first
    bin.install binary => "btyper"
  end

  test do
    assert_match "v#{version}", shell_output("#{bin}/btyper --version")
  end
end
