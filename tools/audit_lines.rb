#!/usr/bin/env ruby
# frozen_string_literal: true

# Audit Go source files for function length and file length.
# Production functions (*.go, excluding *_test.go):
#   - Hard limit: 80 lines
#   - Declarative builders exception: build* with cyclomatic branches <= 2: 150 lines
# Test functions (*_test.go):
#   - Relaxed limit: 200 lines
# File sizing:
#   - Production: comfort 300-700, warn > 800, hard limit 1100
#   - Tests: comfort 300-1000, warn > 1200, hard limit 1600

EXCLUDED_PREFIXES = ["vendor/", ".git/"].freeze

PROD_FUNC_LIMIT = 80
BUILDER_FUNC_LIMIT = 150
TEST_FUNC_LIMIT = 200

PROD_FILE_WARN = 800
PROD_FILE_MAX = 1100
TEST_FILE_WARN = 1200
TEST_FILE_MAX = 1600

def count_branches(lines)
  branches = 0
  in_block_comment = false
  lines.each do |line|
    l = line.dup
    if in_block_comment
      if l =~ %r{\*/}
        l = l.sub(%r{^.*?\*/}, "")
        in_block_comment = false
      else
        next
      end
    end
    l = l.gsub(%r{/\*.*?\*/}, "")
    if l =~ %r{/\*}
      l = l.sub(%r{/\*.*$}, "")
      in_block_comment = true
    end
    l = l.sub(%r{//.*$}, "")
    l = l.gsub(/"(?:[^"\\]|\\.)*"/, '""')
    l = l.gsub(/`[^`]*`/, '``')

    branches += l.scan(/\b(?:if|for|case)\b/).size
    branches += l.scan(/&&|\|\|/).size
  end
  branches
end

files = if ARGV.empty?
          Dir.glob("**/*.go").reject { |f| EXCLUDED_PREFIXES.any? { |p| f.start_with?(p) } }.sort
        else
          ARGV.flat_map { |pattern| Dir.glob(pattern) }.reject { |f| EXCLUDED_PREFIXES.any? { |p| f.start_with?(p) } }.sort
        end

errors = []
warnings = []

files.each do |file|
  next unless File.file?(file)

  is_test = file.end_with?("_test.go")
  lines = File.readlines(file)
  file_len = lines.length

  file_max = is_test ? TEST_FILE_MAX : PROD_FILE_MAX
  file_warn = is_test ? TEST_FILE_WARN : PROD_FILE_WARN

  if file_len > file_max
    errors << "#{file}: file length #{file_len} exceeds hard limit #{file_max}"
  elsif file_len > file_warn
    warnings << "#{file}: file length #{file_len} exceeds comfort warning threshold #{file_warn}"
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
        func_lines = lines[(start_line - 1)..idx]

        if is_test
          limit = TEST_FUNC_LIMIT
          if length > limit
            errors << "#{file}:#{start_line}: test func #{func_name} is #{length} lines (hard limit #{limit})"
          end
        elsif func_name.start_with?("build")
          branches = count_branches(func_lines)
          limit = branches <= 2 ? BUILDER_FUNC_LIMIT : PROD_FUNC_LIMIT
          if length > limit
            errors << "#{file}:#{start_line}: builder func #{func_name} is #{length} lines (hard limit #{limit}, branches=#{branches})"
          end
        else
          limit = PROD_FUNC_LIMIT
          if length > limit
            errors << "#{file}:#{start_line}: func #{func_name} is #{length} lines (hard limit #{limit})"
          end
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
