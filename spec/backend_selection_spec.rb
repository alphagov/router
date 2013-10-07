require 'spec_helper'
require 'net/http'

describe "Selecting a backend based on the routing data" do

  describe "A single path through the router" do
    # This currently relies on data manually added to the database
    before :all do
      @backend = start_test_backend :port => 3160, :identifier => "Fooey"
    end
    after :all do
      stop_test_backend(@backend)
    end

    it "should return the backend response" do
      resp = Net::HTTP.get(URI.parse("http://localhost:3169/foo"))
      expect(resp.strip).to eq("Fooey")
    end
  end
end
