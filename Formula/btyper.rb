class Btyper < Formula
  desc "Adaptive touch-typing practice in the terminal"
  homepage "https://github.com/Ada-lave/btyper"
  version "1.0.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.0/btyper-darwin-amd64", using: :nounzip
      sha256 "f9f86cbbf6ed56868dc34413403dafd38767e3d93942f3d954f0636c0d24e29c"
    end
    on_arm do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.0/btyper-darwin-arm64", using: :nounzip
      sha256 "5dc64eae63c3ef477b67b16768fd4d4cdaaef744b85982a3c11b39d7eb60bfff"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.0/btyper-linux-amd64", using: :nounzip
      sha256 "08848287e18fbc4e6c2dbebb67aef45d9f8e23b7c794f92b9e842ff2b105faa0"
    end
    on_arm do
      url "https://github.com/Ada-lave/btyper/releases/download/v1.0.0/btyper-linux-arm64", using: :nounzip
      sha256 "60a0ff68474c8164107a77122726b0274209755637e732d7026cba92ae044fe3"
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
