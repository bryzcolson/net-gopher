package gopher

import "testing"

func TestItem_String(t *testing.T) {
	tests := []struct {
		name       string
		item       Item
		wantString string
	}{
		{
			name: "happy path",
			item: Item{
				Type:     TypeText,
				Display:  "Hello",
				Selector: "/hello",
				Host:     "localhost",
				Port:     "70",
			},
			wantString: "0Hello\t/hello\tlocalhost\t70\r\n",
		},
		{
			name: "default host",
			item: Item{
				Type:     TypeText,
				Display:  "Default Host",
				Selector: "/default-host",
				Port:     "70",
			},
			wantString: "0Default Host\t/default-host\terror.host\t70\r\n",
		},
		{
			name: "default port",
			item: Item{
				Type:     TypeText,
				Display:  "Default Port",
				Selector: "/default-port",
				Host:     "localhost",
			},
			wantString: "0Default Port\t/default-port\tlocalhost\t70\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.item.String() != tt.wantString {
				t.Errorf("got %q, wanted %q", tt.item.String(), tt.wantString)
			}
		})
	}
}
