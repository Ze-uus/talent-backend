package store

import (
	"errors"
	"net/url"
	"strings"
)

const (
	MaxContentItems  = 50
	MaxContentImages = 20
	MaxContentLinks  = 20
)

func NormalizeAndValidateContent(items []ContentItem) ([]ContentItem, error) {
	if items == nil {
		return []ContentItem{}, nil
	}
	if len(items) > MaxContentItems {
		return nil, errors.New("too_many_content_items")
	}

	out := make([]ContentItem, len(items))
	itemIDs := make(map[string]struct{}, len(items))
	for i, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Title = strings.TrimSpace(item.Title)
		item.Description = strings.TrimSpace(item.Description)
		if item.ID == "" {
			return nil, errors.New("content_item_id_required")
		}
		if _, exists := itemIDs[item.ID]; exists {
			return nil, errors.New("duplicate_content_item_id")
		}
		itemIDs[item.ID] = struct{}{}
		if !plainText(item.Title) || !plainText(item.Description) {
			return nil, errors.New("content_html_not_allowed")
		}
		if len(item.Images) > MaxContentImages {
			return nil, errors.New("too_many_content_images")
		}
		if len(item.Links) > MaxContentLinks {
			return nil, errors.New("too_many_content_links")
		}

		if item.Images == nil {
			item.Images = []string{}
		}
		for j, imageURL := range item.Images {
			imageURL = strings.TrimSpace(imageURL)
			if !validHTTPURL(imageURL) {
				return nil, errors.New("invalid_content_image_url")
			}
			item.Images[j] = imageURL
		}

		if item.Links == nil {
			item.Links = []ContentLink{}
		}
		linkIDs := make(map[string]struct{}, len(item.Links))
		for j, link := range item.Links {
			link.ID = strings.TrimSpace(link.ID)
			link.Label = strings.TrimSpace(link.Label)
			link.URL = strings.TrimSpace(link.URL)
			if link.ID == "" {
				return nil, errors.New("content_link_id_required")
			}
			if _, exists := linkIDs[link.ID]; exists {
				return nil, errors.New("duplicate_content_link_id")
			}
			linkIDs[link.ID] = struct{}{}
			if !plainText(link.Label) {
				return nil, errors.New("content_html_not_allowed")
			}
			if !validHTTPURL(link.URL) {
				return nil, errors.New("invalid_content_link_url")
			}
			item.Links[j] = link
		}
		out[i] = item
	}
	return out, nil
}

func EffectiveContent(campaign Campaign, cycle Cycle) []ContentItem {
	if cycle.Content_override != nil {
		if *cycle.Content_override == nil {
			return []ContentItem{}
		}
		return *cycle.Content_override
	}
	if campaign.Content == nil {
		return []ContentItem{}
	}
	return campaign.Content
}

func validHTTPURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil &&
		(u.Scheme == "http" || u.Scheme == "https") &&
		u.Host != "" &&
		u.User == nil
}

func plainText(value string) bool {
	return !strings.ContainsAny(value, "<>")
}
