#!/usr/bin/env ruby
# frozen_string_literal: true

# Audit Go source files for function length (hard limit 80 lines)
# and file length (comfort 300-700, warn > 800, hard limit 1100).

EXCLUDED_PREFIXES = ["vendor/", ".git/"].freeze
FUNC_LIMIT = 80
FILE_WARN = 800
FILE_MAX = 1100

files = if ARGV.empty?
          Dir.glob("**/*.go").reject { |f| EXCLUDED_PREFIXES.any? { |p| f.start_with?(p) } }.sort
        else
          ARGV.flat_map { |pattern| Dir.glob(pattern) }.reject { |f| EXCLUDED_PREFIXES.any? { |p| f.start_with?(p) } }.sort
        end

errors = []
warnings = []

files.each do |file|
  next unless File.file?(file)

  lines = File.readlines(file)
  file_len = lines.length

  if file_len > FILE_MAX
    errors << "#{file}: file length #{file_len} exceeds hard limit #{FILE_MAX}"
  elsif file_len > FILE_WARN
    warnings << "#{file}: file length #{file_len} exceeds comfort warning threshold #{FILE_WARN}"
  end

  in_func = false
  func_name = ""
  start_line = 0
  brace_depth = 0

  lines.each_with_index do |line, idx|
    if line =~ /^func\s+(?:\([^)]+\)\s+)?([A-Za-z0-9_]+)/ && !in_func
      in_func = true
      func_name = Regexp.last_match(1)
      start_line = idx + 1
      brace_depth = 0
    end

    if in_func
      trimmed = line.strip
      unless trimmed.start_with?("//")
        brace_depth += line.count("{") - line.count("}")
      end

      if brace_depth <= 0 && line.include?("}")
        length = (idx + 1) - start_line + 1
        if length > FUNC_LIMIT
          errors << "#{file}:#{start_line}: func #{func_name} is #{length} lines (hard limit #{FUNC_LIMIT})"
        end
        in_func = false
      end
    end
  end
end

if warnings.any?
  puts "=== Sizing Warnings ==="
  warnings.each { |w| puts "  WARN: #{w}" }
end

if errors.any?
  puts "=== Sizing Errors ==="
  errors.each { |e| puts "  ERROR: #{e}" }
  exit 1
else
  puts "All audited Go files (#{files.size} files) conform to sizing rules."
end
