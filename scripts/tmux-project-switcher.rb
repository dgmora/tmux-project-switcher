#!/usr/bin/ruby
#
# frozen_string_literal: true
#
# @param folder_paths [String] a list of absolute paths pointing to a folder
# @return [Array<String>] a unique list of all projects and tmux sessions
def list
  folders = folder_paths.map { |path| path_to_name(path) }
  sessions = `tmux list-sessions -F '#S'`.split
  (folders + sessions).uniq.sort
end

# @param path [String] An absolute path representing a project folder
# @returns [String] the tmux session name for that project
# @example
#   path_to_name("/Users/david/src/foo/bar/baz")
#     => "bar/baz"
def path_to_name(path)
  path.split('/').last(ENV.fetch('TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT').to_i).join('/')
end

def folder_paths
  return @folder_paths if @folder_paths

  subfolders = '/*' * ENV.fetch('TMUX_PROJECT_SWITCHER_PROJECT_DEPTH').to_i
  @folder_paths = Dir.glob("#{ENV.fetch('TMUX_PROJECT_SWITCHER_ROOT_FOLDER')}#{subfolders}")
end

# @param folder_paths [String] a list of absolute paths pointing to a folder
# @param name [String] Name of the project we are searching
# @return [String] A passed path that matches the name of the project
# @example
#   path(["/Users/david/src/foo/bar", "/Users/david/src/baz/quux"], "foo/bar")
#     => "/Users/david/src/foo/bar"
def path(name)
  folder_paths.detect { |path| path_to_name(path) == name }
end

begin
  case ARGV[0]
  when 'list' then puts(list)
  when 'path' then puts(path(ARGV[1]))
  else puts "'#{ARGV[0]}' is not a command"
  end
rescue StandardError => e
  error = e.backtrace + [e]
  puts error # You get to see something if there's an exception somewhere
end
