class Dvd < Formula
  desc "Bouncing DVD screen saver for your terminal"
  homepage "https://github.com/integrii/dvd"
  license "Unlicense"

  head "https://github.com/integrii/dvd.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/dvd"
  end

  test do
    # Binary should be installed
    assert_predicate bin/"dvd", :exist?
  end
end

