# frozen_string_literal: true

module Abgleich
  class GitHubRepository
    # Owner-wide discovery and repository settings for the fake API.
    module Discovery
      # Notifications carry identity hints; scenarios independently change the
      # state returned by subsequent API reads.
      def repository_notification(action: '', repository: DEFAULT_REPOSITORY, ref: 'refs/heads/master')
        @mutex.synchronize do
          snapshot = @state[:repositories][repository] || repository_data(repository)
          { action:, ref:, installation: { id: INSTALLATION_ID }, repository: JSON.parse(snapshot.to_json, symbolize_names: true) }
        end
      end

      def add_repository(name, **)
        @mutex.synchronize { @state[:repositories][name] = repository_data(name, **) }
      end

      def clear_repositories
        @mutex.synchronize { @state[:repositories].clear }
      end

      def repository(name)
        @mutex.synchronize { @state[:repositories].fetch(name).dup }
      end

      def authenticated_owner=(owner)
        @mutex.synchronize { @state[:login] = owner }
      end

      def authenticated_owner
        @mutex.synchronize { @state[:login] }
      end

      def paginate_repositories
        @mutex.synchronize { @state[:repository_pagination] = true }
      end

      def repositories(page)
        @mutex.synchronize do
          repositories = @state[:repositories].values.sort_by { |repo| repo[:full_name] }
          more = @state[:repository_pagination] && page < repositories.length
          @state[:discovery_complete] = !more
          page_repositories = @state[:repository_pagination] ? repositories.slice(page - 1, 1).to_a : repositories
          [page_repositories.map { |repo| repo.except(:allow_auto_merge, :allow_squash_merge) }, more]
        end
      end

      private

      def repository_data(full_name, **options)
        owner, name = full_name.split('/')
        { name:, full_name:, node_id: "R_#{full_name}", default_branch: 'master', owner: { login: owner },
          archived: options.fetch(:archived, false),
          disabled: options.fetch(:disabled, false), private: options.fetch(:private, false),
          topics: options.fetch(:topics, []),
          allow_auto_merge: options.fetch(:auto_merge, true), allow_squash_merge: options.fetch(:squash_merge, true) }
      end
    end
  end
end
