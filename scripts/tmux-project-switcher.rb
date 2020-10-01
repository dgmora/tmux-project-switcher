# frozen_string_literal: true

#!/usr/bin/ruby

SEPARATOR="\n"

def repos(str)
  puts str.split(SEPARATOR).map { |repo| repo.split("/").last(2).join('/') }.join(SEPARATOR)
end

def paths(str)
  puts full_path
end

def names(str)
  puts str.split(SEPARATOR).map { |repo| repo.split("/").last }.join(SEPARATOR)
end

def diff(str1, str2)
  puts (str1.split(SEPARATOR) - str2.split(SEPARATOR)).join(SEPARATOR)
end

# @param repos_path [String] a list of absolute paths pointing to a repo
# @param sessions [String] a list of open tmux sessions
def list(sessions, repos_paths)
  repos_paths = repos_paths.split(SEPARATOR).sort
  repo_names_to_path = repos_paths.map { |path| [path.split("/").last(2).join("/"), path] }.to_h
  sessions = sessions.split(SEPARATOR).sort
  other_sessions = []
  sessions_for_repos = []
  sessions.each do |s|
    group = repo_names_to_path.key?(s) ? sessions_for_repos : other_sessions
    group << s
  end

  sessions_for_repos_list = sessions_for_repos.map do |s|
    repo_names_to_path[s].split("/").last(2).join("/")
  end
  puts sessions_for_repos_list
  puts other_sessions
  repos_without_session = repo_names_to_path.values.map do |s|
    "#{s.split("/").last(2).join("/")}"
  end
  puts repos_without_session.sort - sessions_for_repos_list
rescue => e
  puts e
end

def path(repos_str, name)
  puts repos_str.split(SEPARATOR).detect { |path| path.end_with?(name) }
end

case ARGV[0]
when "names" then names(ARGV[1])
when "paths" then paths(ARGV[1])
when "repos" then repos(ARGV[1])
when "list" then list(ARGV[1], ARGV[2])
when "diff" then diff(ARGV[1], ARGV[2])
when "path" then path(ARGV[1], ARGV[2])
else puts "'#{ARGV[0]}' is not a command"
end
