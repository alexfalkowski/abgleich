# frozen_string_literal: true

module Abgleich
  # Branch endpoints extend the existing independent fake GitHub API.
  class GitHubService
    get '/repos/:owner/:repository/branches' do
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      page = params.fetch('page', '1').to_i
      halt 503 if Abgleich.github.failure(repository) == 'branch list'
      branches, more = Abgleich.github.branches(repository, page)
      headers 'Link' => "<http://localhost:11001/github/repos/#{repository}/branches?page=#{page + 1}>; rel=\"next\"" if more
      branches.to_json
    end

    get '/repos/:owner/:repository/branches/*' do
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      sleep 1.1 if Abgleich.github.cleanup_mode == 'slow branch'
      Abgleich.github.cleanup_branch(repository, params.fetch('splat').first).to_json
    end

    get '/repos/:owner/:repository/commits/:sha' do
      repository = "#{params.fetch('owner')}/#{params.fetch('repository')}"
      sleep 1.1 if Abgleich.github.cleanup_mode == 'slow commit'
      Abgleich.github.cleanup_commit(repository, params.fetch('sha')).to_json
    end
  end
end
