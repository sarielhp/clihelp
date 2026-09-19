#!/usr/bin/env ruby
# frozen_string_literal: true

# Audit Go source files for function length and file length according to
# /home/sariel/prog/standards/go/GUIDELINES.md (Cognitive Tiering):
#   - Standard Logic: comfort 20-60, soft warn 80, hard limit 110
#   - Declarative Builders (build*, init*, render*, generate*): soft warn 120, hard limit 160 (branches <= 2)
#   - Event / Key Dispatchers (handle*, dispatch*, Execute*): soft warn 150, hard limit 200
#   - Table-Driven Tests: soft warn 180, hard limit 250
# File sizing:
#   - Production: comfort 300-700, warn > 800, hard limit 1100
#   - Tests: comfort 300-1000, warn > 1200, hard limit 1600

EXCLUDED_PREFIXES = ["vendor/", ".git/"].freeze

PROD_FUNC_WARN = 80
PROD_FUNC_MAX = 110

BUILDER_FUNC_WARN = 120
BUILDER_FUNC_MAX = 160

DISPATCHER_FUNC_WARN = 150
DISPATCHER_FUNC_MAX = 200

TEST_FUNC_WARN = 180
TEST_FUNC_MAX = 250

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
          warn_limit = TEST_FUNC_WARN
          hard_limit = TEST_FUNC_MAX
          tier_name = "test func"
        elsif func_name =~ /^(build|init|render|generate|View)/
          branches = count_branches(func_lines)
          warn_limit = BUILDER_FUNC_WARN
          hard_limit = branches <= 2 ? BUILDER_FUNC_MAX : PROD_FUNC_MAX
          tier_name = "builder func"
        elsif func_name =~ /^(handle|dispatch|Execute)/ || func_name.end_with?("Key", "Route")
          warn_limit = DISPATCHER_FUNC_WARN
          hard_limit = DISPATCHER_FUNC_MAX
          tier_name = "dispatcher func"
        else
          warn_limit = PROD_FUNC_WARN
          hard_limit = PROD_FUNC_MAX
          tier_name = "func"
        end

        if length > hard_limit
          errors << "#{file}:#{start_line}: #{tier_name} #{func_name} is #{length} lines (hard limit #{hard_limit})"
        elsif length > warn_limit
          warnings << "#{file}:#{start_line}: #{tier_name} #{func_name} is #{length} lines (soft warn #{warn_limit})"
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
