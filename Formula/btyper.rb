class Btyper < Formula
  desc "Adaptive touch-typing practice in the terminal"
  homepage "https://github.com/Ada-lave/btyper"
  version "0.5.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/Ada-lave/btyper/releases/download/v0.5.0/btyper-darwin-amd64", using: :nounzip
      sha256 "a5608e51744c9914eb544076c1c93912823933f30d0e5241b1acccd63c60b43a"
    end
    on_arm do
      url "https://github.com/Ada-lave/btyper/releases/download/v0.5.0/btyper-darwin-arm64", using: :nounzip
      sha256 "edacdf117b11c9f5fa6baae4f1b5bdbe5ea19c3ba9222ccb03ec8e700d602d73"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/Ada-lave/btyper/releases/download/v0.5.0/btyper-linux-amd64", using: :nounzip
      sha256 "d3f6c2863b7a998d6e993034f65c86d48a0732d3104844af33d608c56299f429"
    end
    on_arm do
      url "https://github.com/Ada-lave/btyper/releases/download/v0.5.0/btyper-linux-arm64", using: :nounzip
      sha256 "5888f8e9dc33d66c0a6e48cd3b46482a87b3742a4cab824fc9248247c2023169"
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
