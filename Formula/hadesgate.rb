# typed: false
# frozen_string_literal: true

class Hadesgate < Formula
  desc "Zero-PII Privacy Gateway & Tactical Shield for LLMs (OpenAI, Anthropic, Gemini)"
  homepage "https://hadesgate.dev"
  version "2.0.0"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/hadesgate/hadesgate/releases/download/v#{version}/hadesgate-darwin-amd64.tar.gz"
    end
    if Hardware::CPU.arm?
      url "https://github.com/hadesgate/hadesgate/releases/download/v#{version}/hadesgate-darwin-arm64.tar.gz"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/hadesgate/hadesgate/releases/download/v#{version}/hadesgate-linux-amd64.tar.gz"
    end
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/hadesgate/hadesgate/releases/download/v#{version}/hadesgate-linux-arm64.tar.gz"
    end
  end

  def install
    bin.install "hadesgate"
    etc.install ".env.example" => "hadesgate.env.example" if File.exist?(".env.example")
  end

  def post_install
    (var/"hadesgate").mkpath
    (var/"log").mkpath
  end

  service do
    run [opt_bin/"hadesgate"]
    keep_alive true
    working_dir var/"hadesgate"
    log_path var/"log/hadesgate.log"
    error_log_path var/"log/hadesgate.log"
    environment_variables HADESGATE_PORT: "8080"
  end

  test do
    # Verify binary exists and executes
    assert_predicate bin/"hadesgate", :exist?
    assert_predicate bin/"hadesgate", :executable?
  end
end

