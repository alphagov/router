require 'spec_helper'

describe "error handling" do

  describe "handling a panic" do
    before :each do
      add_route "/boom", :handler => "boom"
      reload_routes
    end

    it "should return a 500 error to the client" do
      response = router_request("/boom")
      expect(response.code).to eq(500)
    end

    it "should log the fact"
  end
end
